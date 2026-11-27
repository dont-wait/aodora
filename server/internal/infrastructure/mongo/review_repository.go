package mongo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	app "github.com/your-org/aodora/server/internal/application/review"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const displayFallback = "Khách hàng AN SOIE"

type ReviewRepository struct {
	client    *driver.Client
	reviews   *driver.Collection
	customers *driver.Collection
	products  *driver.Collection
}

func NewReviewRepository(db *driver.Database) *ReviewRepository {
	return &ReviewRepository{client: db.Client(), reviews: db.Collection("reviews"), customers: db.Collection("customers"), products: db.Collection("products")}
}

func (r *ReviewRepository) List(ctx context.Context, productID string, page, size int64) (app.Page, error) {
	productOID, err := bson.ObjectIDFromHex(productID)
	visible, visibilityErr := r.publicProduct(ctx, productOID)
	if visibilityErr != nil {
		return app.Page{}, fmt.Errorf("check review product: %w", visibilityErr)
	}
	if err != nil || !visible {
		return app.Page{}, app.ErrProduct
	}
	filter := bson.M{"product_id": productOID}
	count, average, err := reviewStats(ctx, r.reviews, productOID)
	if err != nil {
		return app.Page{}, err
	}
	cur, err := r.reviews.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetSkip((page-1)*size).SetLimit(size))
	if err != nil {
		return app.Page{}, fmt.Errorf("list reviews: %w", err)
	}
	defer cur.Close(ctx)
	items := make([]app.PublicReview, 0)
	for cur.Next(ctx) {
		var row bson.M
		if err := cur.Decode(&row); err != nil {
			return app.Page{}, err
		}
		items = append(items, publicReview(row))
	}
	if err := cur.Err(); err != nil {
		return app.Page{}, err
	}
	return app.Page{Items: items, Total: count, Page: page, Size: size, Average: average}, nil
}

func (r *ReviewRepository) Own(ctx context.Context, customerID, productID string) (*app.OwnReview, error) {
	customerOID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		return nil, app.ErrNotFound
	}
	productOID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return nil, app.ErrProduct
	}
	var row bson.M
	err = r.reviews.FindOne(ctx, bson.M{"customer_id": customerOID, "product_id": productOID}).Decode(&row)
	if errors.Is(err, driver.ErrNoDocuments) {
		return nil, app.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find own review: %w", err)
	}
	own := ownReview(row)
	return &own, nil
}

// Upsert keeps one review per account and product (uq_reviews_customer_product).
func (r *ReviewRepository) Upsert(ctx context.Context, customerID, productID string, input app.Input) (app.OwnReview, error) {
	customerOID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		return app.OwnReview{}, app.ErrNotFound
	}
	productOID, err := r.visibleProductID(ctx, productID)
	if err != nil {
		return app.OwnReview{}, err
	}
	orderOID, err := bson.ObjectIDFromHex(input.OrderID)
	if err != nil {
		return app.OwnReview{}, app.Invalid{Field: "order_id", Message: "ID đơn hàng không hợp lệ"}
	}
	var customer struct {
		FullName string `bson:"full_name"`
	}
	if err := r.customers.FindOne(ctx, bson.M{"_id": customerOID}).Decode(&customer); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return app.OwnReview{}, app.ErrNotFound
		}
		return app.OwnReview{}, fmt.Errorf("find review customer: %w", err)
	}
	filter := bson.M{"customer_id": customerOID, "product_id": productOID}
	return r.upsert(ctx, filter, productOID, orderOID, displayOrFallback(customer.FullName), input.Rating, input.Content, input.Images)
}

// Statuses returns one entry per requested product, in request order.
// Unknown or non-public products get an empty slug; invalid IDs are kept
// with an empty slug so the caller can still render the order line.
func (r *ReviewRepository) Statuses(ctx context.Context, customerID string, productIDs []string) ([]app.ProductReviewStatus, error) {
	result := make([]app.ProductReviewStatus, len(productIDs))
	oids := make([]bson.ObjectID, 0, len(productIDs))
	for i, id := range productIDs {
		result[i].ProductID = id
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			oids = append(oids, oid)
		}
	}
	if len(oids) == 0 {
		return result, nil
	}
	pipeline := bson.A{bson.M{"$match": bson.M{"_id": bson.M{"$in": oids}, "is_active": true}}}
	pipeline = append(pipeline, categoryVisibilityStages()...)
	pipeline = append(pipeline, bson.M{"$project": bson.M{"slug": 1}})
	cur, err := r.products.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("read review product slugs: %w", err)
	}
	slugs := map[string]string{}
	for cur.Next(ctx) {
		var row struct {
			ID   bson.ObjectID `bson:"_id"`
			Slug string        `bson:"slug"`
		}
		if err := cur.Decode(&row); err != nil {
			_ = cur.Close(ctx)
			return nil, err
		}
		slugs[row.ID.Hex()] = row.Slug
	}
	if err := cur.Err(); err != nil {
		_ = cur.Close(ctx)
		return nil, err
	}
	_ = cur.Close(ctx)
	reviewed := map[string]bool{}
	if customerOID, err := bson.ObjectIDFromHex(customerID); err == nil {
		cur, err := r.reviews.Find(ctx, bson.M{"customer_id": customerOID, "product_id": bson.M{"$in": oids}}, options.Find().SetProjection(bson.M{"product_id": 1}))
		if err != nil {
			return nil, fmt.Errorf("read reviewed products: %w", err)
		}
		defer cur.Close(ctx)
		for cur.Next(ctx) {
			var row struct {
				ProductID bson.ObjectID `bson:"product_id"`
			}
			if err := cur.Decode(&row); err != nil {
				return nil, err
			}
			reviewed[row.ProductID.Hex()] = true
		}
		if err := cur.Err(); err != nil {
			return nil, err
		}
	}
	for i := range result {
		result[i].Slug = slugs[result[i].ProductID]
		result[i].Reviewed = reviewed[result[i].ProductID]
	}
	return result, nil
}

func (r *ReviewRepository) AdminList(ctx context.Context, filter app.AdminFilter) (app.AdminPage, error) {
	match := bson.M{}
	if filter.ProductID != "" {
		productOID, err := bson.ObjectIDFromHex(filter.ProductID)
		if err != nil {
			return app.AdminPage{}, app.Invalid{Field: "product_id", Message: "ID sản phẩm không hợp lệ"}
		}
		match["product_id"] = productOID
	}
	if filter.Rating != 0 {
		match["rating"] = filter.Rating
	}
	total, err := r.reviews.CountDocuments(ctx, match)
	if err != nil {
		return app.AdminPage{}, fmt.Errorf("count admin reviews: %w", err)
	}
	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$sort": bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}},
		bson.M{"$skip": (filter.Page - 1) * filter.Size},
		bson.M{"$limit": filter.Size},
		bson.M{"$lookup": bson.M{"from": "products", "localField": "product_id", "foreignField": "_id", "as": "product",
			"pipeline": bson.A{bson.M{"$project": bson.M{"name": 1, "slug": 1}}}}},
		bson.M{"$lookup": bson.M{"from": "orders", "localField": "order_id", "foreignField": "_id", "as": "order",
			"pipeline": bson.A{bson.M{"$project": bson.M{"order_number": 1}}}}},
	}
	cur, err := r.reviews.Aggregate(ctx, pipeline)
	if err != nil {
		return app.AdminPage{}, fmt.Errorf("list admin reviews: %w", err)
	}
	defer cur.Close(ctx)
	items := make([]app.AdminReview, 0)
	for cur.Next(ctx) {
		var row bson.M
		if err := cur.Decode(&row); err != nil {
			return app.AdminPage{}, err
		}
		item := app.AdminReview{PublicReview: publicReview(row), ProductID: asID(row["product_id"]), UpdatedAt: asTime(row["updated_at"]).UTC().Format(time.RFC3339)}
		if products := asSlice(row["product"]); len(products) > 0 {
			product := asMap(products[0])
			item.ProductName, item.ProductSlug = asString(product["name"]), asString(product["slug"])
		}
		if orders := asSlice(row["order"]); len(orders) > 0 {
			item.OrderNumber = asString(asMap(orders[0])["order_number"])
		}
		items = append(items, item)
	}
	if err := cur.Err(); err != nil {
		return app.AdminPage{}, err
	}
	return app.AdminPage{Items: items, Total: total, Page: filter.Page, Size: filter.Size}, nil
}

// Delete removes the review and recomputes the product aggregate atomically.
func (r *ReviewRepository) Delete(ctx context.Context, id string) error {
	reviewOID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return app.ErrNotFound
	}
	return r.transaction(ctx, func(tx context.Context) error {
		var removed struct {
			ProductID bson.ObjectID `bson:"product_id"`
		}
		err := r.reviews.FindOneAndDelete(tx, bson.M{"_id": reviewOID}, options.FindOneAndDelete().SetProjection(bson.M{"product_id": 1})).Decode(&removed)
		if errors.Is(err, driver.ErrNoDocuments) {
			return app.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("delete review: %w", err)
		}
		return r.refreshSummary(tx, removed.ProductID)
	})
}

func (r *ReviewRepository) upsert(ctx context.Context, filter bson.M, productOID, orderOID bson.ObjectID, displayName string, rating int64, content string, images []app.Image) (app.OwnReview, error) {
	if images == nil {
		images = []app.Image{}
	}
	write := func() error {
		return r.transaction(ctx, func(tx context.Context) error {
			now := time.Now().UTC()
			_, err := r.reviews.UpdateOne(tx, filter, bson.M{
				"$set":         bson.M{"order_id": orderOID, "rating": rating, "content": content, "images": images, "display_name": displayName, "verified_purchase": true, "updated_at": now},
				"$setOnInsert": bson.M{"created_at": now},
			}, options.UpdateOne().SetUpsert(true))
			if err != nil {
				return err
			}
			return r.refreshSummary(tx, productOID)
		})
	}
	err := write()
	// Two first submissions can race on the unique index; the loser retries
	// once and then matches the winner's document instead of inserting.
	if driver.IsDuplicateKeyError(err) {
		err = write()
	}
	if err != nil {
		return app.OwnReview{}, fmt.Errorf("upsert review: %w", err)
	}
	var row bson.M
	if err := r.reviews.FindOne(ctx, filter).Decode(&row); err != nil {
		return app.OwnReview{}, fmt.Errorf("read saved review: %w", err)
	}
	return ownReview(row), nil
}

// transaction makes the review write and the aggregate update one unit.
// Concurrent writers for the same product conflict on the product document;
// WithTransaction retries the loser, so the stored aggregate cannot go stale.
func (r *ReviewRepository) transaction(ctx context.Context, fn func(context.Context) error) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("start review session: %w", err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		return nil, fn(tx)
	})
	return err
}

func (r *ReviewRepository) refreshSummary(ctx context.Context, productOID bson.ObjectID) error {
	count, average, err := reviewStats(ctx, r.reviews, productOID)
	if err != nil {
		return err
	}
	if _, err := r.products.UpdateOne(ctx, bson.M{"_id": productOID}, bson.M{"$set": bson.M{"rating": average, "review_count": count}}); err != nil {
		return fmt.Errorf("update product review summary: %w", err)
	}
	return nil
}

// reviewStats is the single definition of the public aggregate: every
// stored review counts, the average is rounded to two decimals, and no
// reviews means 0/0.
func reviewStats(ctx context.Context, reviews *driver.Collection, productOID bson.ObjectID) (int64, float64, error) {
	cur, err := reviews.Aggregate(ctx, bson.A{
		bson.M{"$match": bson.M{"product_id": productOID}},
		bson.M{"$group": bson.M{"_id": nil, "count": bson.M{"$sum": 1}, "average": bson.M{"$avg": "$rating"}}},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("aggregate reviews: %w", err)
	}
	defer cur.Close(ctx)
	if !cur.Next(ctx) {
		return 0, 0, cur.Err()
	}
	var row bson.M
	if err := cur.Decode(&row); err != nil {
		return 0, 0, fmt.Errorf("decode review stats: %w", err)
	}
	return asInt64(row["count"]), math.Round(asFloat64(row["average"])*100) / 100, nil
}

func (r *ReviewRepository) visibleProductID(ctx context.Context, productID string) (bson.ObjectID, error) {
	productOID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return bson.ObjectID{}, app.ErrProduct
	}
	visible, err := r.publicProduct(ctx, productOID)
	if err != nil {
		return bson.ObjectID{}, fmt.Errorf("check review product: %w", err)
	}
	if !visible {
		return bson.ObjectID{}, app.ErrProduct
	}
	return productOID, nil
}

func displayOrFallback(name string) string {
	if name == "" {
		return displayFallback
	}
	return name
}

func ownReview(row bson.M) app.OwnReview {
	return app.OwnReview{PublicReview: publicReview(row), OrderID: asID(row["order_id"])}
}

func publicReview(row bson.M) app.PublicReview {
	images := make([]app.Image, 0)
	for _, raw := range asSlice(row["images"]) {
		image := asMap(raw)
		images = append(images, app.Image{Src: asString(image["src"]), Alt: asString(image["alt"])})
	}
	content := asString(row["content"])
	if content == "" {
		content = asString(row["comment"])
	}
	return app.PublicReview{ID: asID(row["_id"]), Rating: asInt64(row["rating"]), Content: content, DisplayName: displayName(row), Images: images, VerifiedPurchase: asBool(row["verified_purchase"]), CreatedAt: asTime(row["created_at"]).UTC().Format(time.RFC3339)}
}

func displayName(row bson.M) string {
	return displayOrFallback(asString(row["display_name"]))
}

func (r *ReviewRepository) publicProduct(ctx context.Context, id bson.ObjectID) (bool, error) {
	pipeline := bson.A{bson.M{"$match": bson.M{"_id": id, "is_active": true}}}
	pipeline = append(pipeline, categoryVisibilityStages()...)
	pipeline = append(pipeline, bson.M{"$limit": 1})
	cur, err := r.products.Aggregate(ctx, pipeline)
	if err != nil {
		return false, err
	}
	defer cur.Close(ctx)
	if !cur.Next(ctx) {
		return false, cur.Err()
	}
	return true, nil
}
