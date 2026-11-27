// Command seedorders fills the dev database with about a week of plausible
// purchase history (Vietnamese customers, orders spread over shop hours) so the
// admin dashboard has data. It only touches documents tagged seed_tag=order-history.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/your-org/aodora/server/internal/config"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seedorders:", err)
		os.Exit(1)
	}
}

func run() error {
	from := flag.String("from", "2026-11-21", "first day of history (YYYY-MM-DD, Vietnam time)")
	seed := flag.Uint64("seed", 20261121, "random seed; same seed and window give the same data")
	customers := flag.Int("customers", 40, "number of demo customers")
	divisions := flag.String("divisions", "../client/src/data/vn-divisions-2025.json", "province/ward list (2-level, 2025)")
	purge := flag.Bool("purge", false, "only remove previously seeded order history")
	flag.Parse()
	if err := config.LoadEnv(); err != nil {
		return err
	}
	start, err := time.ParseInLocation("2006-01-02", *from, vnZone)
	if err != nil {
		return fmt.Errorf("-from: %w", err)
	}
	codFee, err := codFeeFromEnv()
	if err != nil {
		return err
	}
	wards, err := loadWards(*divisions)
	if err != nil && !*purge {
		return err
	}

	mongoConfig, err := config.LoadMongoConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db, err := database.OpenMongo(ctx, database.Config{URI: mongoConfig.URI, DB: mongoConfig.DB})
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(context.Background()) }()

	removedOrders, err := db.DB.Collection("orders").DeleteMany(ctx, bson.M{"seed_tag": seedTag})
	if err != nil {
		return err
	}
	removedCustomers, err := db.DB.Collection("customers").DeleteMany(ctx, bson.M{"seed_tag": seedTag})
	if err != nil {
		return err
	}
	fmt.Printf("removed previous batch: %d orders, %d customers\n", removedOrders.DeletedCount, removedCustomers.DeletedCount)
	if *purge {
		return nil
	}

	products, err := loadProducts(ctx, db)
	if err != nil {
		return err
	}
	if codFee == nil {
		fmt.Println("ORDER_COD_SHIPPING_FEE is blank (Q-04): generating cash-at-counter orders only")
	}
	b, err := generate(options{From: start, Now: time.Now(), Seed: *seed, Customers: *customers, CODFee: codFee}, products, wards)
	if err != nil {
		return err
	}
	if len(b.Customers) > 0 {
		if _, err := db.DB.Collection("customers").InsertMany(ctx, b.Customers); err != nil {
			return fmt.Errorf("insert customers: %w", err)
		}
	}
	if len(b.Orders) > 0 {
		if _, err := db.DB.Collection("orders").InsertMany(ctx, b.Orders); err != nil {
			return fmt.Errorf("insert orders: %w", err)
		}
	}
	fmt.Printf("seeded %d customers and %d orders from %s to now\n", len(b.Customers), len(b.Orders), start.Format("02/01/2006"))
	return nil
}

func codFeeFromEnv() (*int64, error) {
	raw := strings.TrimSpace(os.Getenv("ORDER_COD_SHIPPING_FEE"))
	if raw == "" {
		return nil, nil
	}
	fee, err := domainorder.ParseMoney(raw)
	if err != nil {
		return nil, fmt.Errorf("ORDER_COD_SHIPPING_FEE is invalid: %w", err)
	}
	v, ok := fee.Int64()
	if !ok {
		return nil, errors.New("ORDER_COD_SHIPPING_FEE must be a whole number of đồng")
	}
	return &v, nil
}

func loadWards(path string) ([]ward, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file struct {
		Provinces []struct {
			Name  string `json:"name"`
			Wards []struct {
				Name string `json:"name"`
			} `json:"wards"`
		} `json:"provinces"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	var out []ward
	for _, p := range file.Provinces {
		for _, w := range p.Wards {
			out = append(out, ward{Name: w.Name, Province: p.Name})
		}
	}
	return out, nil
}

// loadProducts returns storefront-visible products (active, in an active
// category) that have at least one SKU, which is what checkout can sell.
func loadProducts(ctx context.Context, db *database.Mongo) ([]product, error) {
	cursor, err := db.DB.Collection("products").Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"is_active": true, "variants.sku": bson.M{"$exists": true}}},
		bson.M{"$lookup": bson.M{"from": "categories", "localField": "category_id", "foreignField": "_id", "as": "_category"}},
		bson.M{"$match": bson.M{"_category.is_active": true}},
		bson.M{"$project": bson.M{"name": 1, "price": 1, "variants": 1}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var out []product
	for cursor.Next(ctx) {
		var doc struct {
			ID       bson.ObjectID `bson:"_id"`
			Name     string        `bson:"name"`
			Price    int64         `bson:"price"`
			Variants []struct {
				SKU   string `bson:"sku"`
				Size  string `bson:"size"`
				Color string `bson:"color"`
			} `bson:"variants"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue // price stored as Decimal128 or malformed: skip rather than guess
		}
		p := product{ID: doc.ID, Name: doc.Name, Price: doc.Price}
		for _, v := range doc.Variants {
			if v.SKU != "" {
				p.Variants = append(p.Variants, variant{SKU: v.SKU, Size: v.Size, Color: v.Color})
			}
		}
		if len(p.Variants) > 0 && p.Price > 0 {
			out = append(out, p)
		}
	}
	return out, cursor.Err()
}
