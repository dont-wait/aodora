// Command seed adds opt-in demo data without replacing existing records.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/database"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

type category struct {
	Key, Name, Slug string
}

type picture struct {
	File      string `json:"file"`
	Alt       string `json:"alt"`
	SourceURL string `json:"source_url"`
}

type product struct {
	Key, Category, Name, Slug, Description, Material, Color, Hex string
	SourceURL                                                    string `json:"source_url"`
	Price                                                        int64
	Sizes                                                        []string
	Stocks                                                       []int64
	Images                                                       []picture
}

type manifest struct {
	Source     string
	Categories []category
	Products   []product
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	manifestPath := flag.String("manifest", "seeds/forher.json", "curated catalog manifest")
	assetsPath := flag.String("assets", "seeds/assets", "downloaded images directory")
	checkOnly := flag.Bool("check", false, "validate manifest and images without database writes")
	accountsOnly := flag.Bool("accounts-only", false, "only create the demo admin/customer accounts (no catalog, no images)")
	flag.Parse()
	if err := config.LoadEnv(); err != nil {
		return err
	}
	if *accountsOnly {
		return withMongo(seedAccounts)
	}
	data, err := os.ReadFile(*manifestPath)
	if err != nil {
		return err
	}
	var seed manifest
	if err := json.Unmarshal(data, &seed); err != nil {
		return err
	}
	baseURL := publicURL()
	if err := validate(seed, baseURL, *assetsPath); err != nil {
		return err
	}
	if *checkOnly {
		fmt.Printf("Validated %d categories, %d products and their images\n", len(seed.Categories), len(seed.Products))
		return nil
	}
	return withMongo(func(ctx context.Context, db *database.Mongo) error {
		if err := seedCatalog(ctx, db, seed, baseURL); err != nil {
			return err
		}
		return seedAccounts(ctx, db)
	})
}

// withMongo opens the configured database for one seeding run.
func withMongo(fn func(context.Context, *database.Mongo) error) error {
	mongoConfig, err := config.LoadMongoConfig()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db, err := database.OpenMongo(ctx, database.Config{URI: mongoConfig.URI, DB: mongoConfig.DB})
	if err != nil {
		return err
	}
	defer func() { _ = db.Close(context.Background()) }()
	return fn(ctx, db)
}

func seedCatalog(ctx context.Context, db *database.Mongo, seed manifest, baseURL string) error {
	for _, name := range []string{"categories", "products"} {
		collections, err := db.DB.ListCollectionNames(ctx, bson.M{"name": name})
		if err != nil {
			return err
		}
		if len(collections) == 0 {
			return fmt.Errorf("missing %s collection: run make migrate-up first", name)
		}
	}
	categories, products := documents(seed, baseURL, time.Now().UTC())
	for _, group := range []struct {
		name string
		docs []bson.M
	}{{"categories", categories}, {"products", products}} {
		inserted := int64(0)
		for _, doc := range group.docs {
			result, err := db.DB.Collection(group.name).UpdateOne(ctx,
				bson.M{"_id": doc["_id"]}, bson.M{"$setOnInsert": doc}, options.UpdateOne().SetUpsert(true))
			if err != nil {
				return fmt.Errorf("seed %s %s: %w", group.name, doc["slug"], err)
			}
			inserted += result.UpsertedCount
		}
		fmt.Printf("%s: inserted %d, preserved %d\n", group.name, inserted, int64(len(group.docs))-inserted)
	}
	return nil
}

// seedAccounts upserts the demo accounts; existing accounts with the same id or email are left untouched.
func seedAccounts(ctx context.Context, db *database.Mongo) error {
	customers, err := db.DB.ListCollectionNames(ctx, bson.M{"name": "customers"})
	if err != nil {
		return err
	}
	if len(customers) == 0 {
		return fmt.Errorf("missing customers collection: run make migrate-up first")
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password@123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	authUsers := authDocuments(string(passwordHash), time.Now().UTC())
	inserted := int64(0)
	for _, doc := range authUsers {
		result, err := db.DB.Collection("customers").UpdateOne(ctx, bson.M{"$or": bson.A{
			bson.M{"_id": doc["_id"]}, bson.M{"email_normalized": doc["email_normalized"]}, bson.M{"email": doc["email"]},
		}}, bson.M{"$setOnInsert": doc}, options.UpdateOne().SetUpsert(true))
		if err != nil {
			return fmt.Errorf("seed customer %s: %w", doc["email"], err)
		}
		inserted += result.UpsertedCount
	}
	fmt.Printf("customers: inserted %d, preserved %d\n", inserted, int64(len(authUsers))-inserted)
	return nil
}

func publicURL() string {
	if value := strings.TrimSpace(os.Getenv("RUSTFS_PUBLIC_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	port, bucket := os.Getenv("RUSTFS_API_PORT"), os.Getenv("RUSTFS_BUCKET")
	if port == "" {
		port = "9100"
	}
	if bucket == "" {
		bucket = "aodora-images"
	}
	return "http://localhost:" + port + "/" + bucket
}

func validate(seed manifest, baseURL, assetsPath string) error {
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("RUSTFS_PUBLIC_URL must be a public HTTP(S) bucket URL")
	}
	categories := map[string]bool{}
	for _, c := range seed.Categories {
		if c.Key == "" || c.Name == "" || c.Slug == "" || categories[c.Key] {
			return fmt.Errorf("invalid or duplicate category %q", c.Key)
		}
		categories[c.Key] = true
	}
	keys := map[string]bool{}
	for _, p := range seed.Products {
		if !categories[p.Category] || p.Key == "" || keys[p.Key] || p.Name == "" || p.Slug == "" || p.Price < 0 || len(p.Sizes) == 0 || len(p.Sizes) != len(p.Stocks) || len(p.Images) < 2 {
			return fmt.Errorf("invalid demo product %q", p.Key)
		}
		keys[p.Key] = true
		for _, n := range p.Stocks {
			if n < 0 {
				return fmt.Errorf("negative stock for %s", p.Key)
			}
		}
		for _, image := range p.Images {
			if image.Alt == "" || filepath.IsAbs(image.File) || !filepath.IsLocal(image.File) || filepath.Ext(image.File) != ".webp" {
				return fmt.Errorf("invalid image path or alt: %q", image.File)
			}
			data, err := os.ReadFile(filepath.Join(assetsPath, image.File))
			if err != nil {
				return fmt.Errorf("image %s missing: run make seed-fetch: %w", image.File, err)
			}
			if len(data) <= 12 || len(data) > 20*1024*1024 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
				return fmt.Errorf("invalid WebP image: %s", image.File)
			}
		}
	}
	if len(seed.Categories) == 0 || len(seed.Products) == 0 {
		return fmt.Errorf("empty demo manifest")
	}
	return nil
}

func demoID(kind, key string) bson.ObjectID {
	hash := sha256.Sum256([]byte("aodora:demo:forher:" + kind + ":" + key))
	return bson.ObjectID(hash[:12])
}

func authDocuments(passwordHash string, now time.Time) []bson.M {
	return []bson.M{
		{
			"_id": demoID("customer", "admin"), "full_name": "Aodora Admin", "email": "admin@aodora.local",
			"email_normalized": "admin@aodora.local", "password_hash": passwordHash, "role": "admin", "phone": "0900000001",
			"created_at": now, "updated_at": now,
		},
		{
			"_id": demoID("customer", "user"), "full_name": "Aodora Demo User", "email": "user@aodora.local",
			"email_normalized": "user@aodora.local", "password_hash": passwordHash, "role": "customer", "phone": "0900000002",
			"created_at": now, "updated_at": now,
		},
	}
}

func documents(seed manifest, baseURL string, now time.Time) ([]bson.M, []bson.M) {
	categories, products := []bson.M{}, []bson.M{}
	for i, c := range seed.Categories {
		categories = append(categories, bson.M{
			"_id": demoID("category", c.Key), "name": c.Name, "slug": c.Slug,
			"description": "Danh mục mẫu phục vụ thử nghiệm Aodora.", "parent_id": nil,
			"is_active": true, "sort_order": int64(i + 1), "version": int64(1), "created_at": now, "updated_at": now,
		})
	}
	for _, p := range seed.Products {
		images, urls, variants := []bson.M{}, []string{}, []bson.M{}
		for _, image := range p.Images {
			src := baseURL + "/demo/forher/" + image.File
			images = append(images, bson.M{"src": src, "alt": image.Alt})
			urls = append(urls, src)
		}
		stock := int64(0)
		for i, size := range p.Sizes {
			variants = append(variants, bson.M{"sku": "DEMO-" + strings.ToUpper(p.Key) + "-" + size, "size": size, "color": p.Color, "stock": p.Stocks[i]})
			stock += p.Stocks[i]
		}
		products = append(products, bson.M{
			"_id": demoID("product", p.Key), "category_id": demoID("category", p.Category),
			"name": p.Name, "slug": p.Slug, "description": p.Description, "material": p.Material,
			"price": p.Price, "compare_at_price": nil, "stock": stock, "variants": variants,
			"images": urls, "image": images[0], "gallery": images[1:], "colors": []bson.M{{"name": p.Color, "hex": p.Hex}},
			"sizes": p.Sizes, "is_active": true, "bespoke": true, "collection": "Demo tham khảo Forher",
			"source_url": p.SourceURL, "seed_source": seed.Source,
			"story":       bson.M{"eyebrow": "Bộ sưu tập tham khảo", "title": p.Name, "paragraphs": []string{p.Description}, "image": images[0]},
			"fabric":      bson.M{"title": p.Material, "description": "Thông tin chất liệu tham khảo từ trang sản phẩm nguồn.", "image": images[1], "care": []bson.M{}},
			"accessories": []bson.M{}, "trust": []bson.M{},
			"version": int64(1), "created_at": now, "updated_at": now,
		})
	}
	return categories, products
}
