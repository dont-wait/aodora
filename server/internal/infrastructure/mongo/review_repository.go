package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	app "github.com/your-org/aodora/server/internal/application/review"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ReviewRepository struct {
	reviews    *driver.Collection
	customers  *driver.Collection
	products   *driver.Collection
	categories *driver.Collection
}

func NewReviewRepository(db *driver.Database) *ReviewRepository {
	return &ReviewRepository{reviews: db.Collection("reviews"), customers: db.Collection("customers"), products: db.Collection("products"), categories: db.Collection("categories")}
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
	total, err := r.reviews.CountDocuments(ctx, filter)
	if err != nil {
		return app.Page{}, fmt.Errorf("count reviews: %w", err)
	}
	average := float64(0)
	cur, err := r.reviews.Aggregate(ctx, bson.A{bson.M{"$match": filter}, bson.M{"$group": bson.M{"_id": nil, "average": bson.M{"$avg": "$rating"}}}})
	if err != nil {
		return app.Page{}, fmt.Errorf("aggregate reviews: %w", err)
	}
	if cur.Next(ctx) {
		var row bson.M
		if err := cur.Decode(&row); err != nil {
			_ = cur.Close(ctx)
			return app.Page{}, err
		}
		average = asFloat64(row["average"])
	}
	_ = cur.Close(ctx)
	if total > 0 {
		average = float64(int(average*100+0.5)) / 100
	}
	cur, err = r.reviews.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).SetSkip((page-1)*size).SetLimit(size))
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
	return app.Page{Items: items, Total: total, Page: page, Size: size, Average: average}, nil
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
	public := publicReview(row)
	return &app.OwnReview{PublicReview: public, OrderID: asID(row["order_id"])}, nil
}

func (r *ReviewRepository) Upsert(ctx context.Context, customerID, productID string, input app.Input) (app.OwnReview, error) {
	customerOID, err := bson.ObjectIDFromHex(customerID)
	if err != nil {
		return app.OwnReview{}, app.ErrNotFound
	}
	productOID, err := bson.ObjectIDFromHex(productID)
	visible, visibilityErr := r.publicProduct(ctx, productOID)
	if visibilityErr != nil {
		return app.OwnReview{}, fmt.Errorf("check review product: %w", visibilityErr)
	}
	if err != nil || !visible {
		return app.OwnReview{}, app.ErrProduct
	}
	orderOID, err := bson.ObjectIDFromHex(input.OrderID)
	if err != nil {
		return app.OwnReview{}, app.Invalid{Field: "order_id", Message: "ID đơn hàng không hợp lệ"}
	}
	var customer struct {
		FullName string `bson:"full_name"`
	}
	if err := r.customers.FindOne(ctx, bson.M{"_id": customerOID}).Decode(&customer); err != nil {
		return app.OwnReview{}, app.ErrNotFound
	}
	now := time.Now().UTC()
	_, err = r.reviews.UpdateOne(ctx, bson.M{"customer_id": customerOID, "product_id": productOID}, bson.M{
		"$set":         bson.M{"order_id": orderOID, "rating": input.Rating, "content": input.Content, "images": input.Images, "display_name": customer.FullName, "verified_purchase": true, "updated_at": now},
		"$setOnInsert": bson.M{"created_at": now},
	}, options.UpdateOne().SetUpsert(true))
	if driver.IsDuplicateKeyError(err) {
		return app.OwnReview{}, fmt.Errorf("upsert review: %w", err)
	}
	if err != nil {
		return app.OwnReview{}, fmt.Errorf("upsert review: %w", err)
	}
	result, err := r.Own(ctx, customerID, productID)
	if err != nil || result == nil {
		return app.OwnReview{}, err
	}
	return *result, nil
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
	if value := asString(row["display_name"]); value != "" {
		return value
	}
	return "Khách hàng AN SOIE"
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
