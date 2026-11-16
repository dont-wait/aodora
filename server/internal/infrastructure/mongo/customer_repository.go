package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	app "github.com/your-org/aodora/server/internal/application/customer"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type CustomerRepository struct {
	customers  *driver.Collection
	products   *driver.Collection
	categories *driver.Collection
}

func NewCustomerRepository(db *driver.Database) *CustomerRepository {
	return &CustomerRepository{
		customers: db.Collection("customers"), products: db.Collection("products"), categories: db.Collection("categories"),
	}
}

type customerProfileDocument struct {
	ID               bson.ObjectID     `bson:"_id"`
	FullName         string            `bson:"full_name"`
	Email            string            `bson:"email"`
	Phone            string            `bson:"phone,omitempty"`
	Addresses        []app.Address     `bson:"addresses,omitempty"`
	Measurements     *app.Measurements `bson:"measurement_profile,omitempty"`
	MeasurementNotes string            `bson:"measurement_notes,omitempty"`
	WishlistIDs      []bson.ObjectID   `bson:"wishlist_product_ids,omitempty"`
	Version          int64             `bson:"profile_version,omitempty"`
}

func (r *CustomerRepository) Profile(ctx context.Context, userID string) (app.Profile, error) {
	id, err := customerObjectID(userID)
	if err != nil {
		return app.Profile{}, err
	}
	var doc customerProfileDocument
	err = r.customers.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.Profile{}, app.ErrNotFound
	}
	if err != nil {
		return app.Profile{}, fmt.Errorf("find customer profile: %w", err)
	}
	return profileFromDocument(doc), nil
}

func profileFromDocument(doc customerProfileDocument) app.Profile {
	addresses := append([]app.Address(nil), doc.Addresses...)
	if addresses == nil {
		addresses = []app.Address{}
	}
	for i := range addresses {
		if addresses[i].ID == "" {
			addresses[i].ID = fmt.Sprintf("legacy-%d", i)
		}
	}
	return app.Profile{ID: doc.ID.Hex(), FullName: doc.FullName, Email: doc.Email, Phone: doc.Phone, Addresses: addresses, Measurements: doc.Measurements, MeasurementNotes: doc.MeasurementNotes, Version: doc.Version}
}

func customerObjectID(value string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		return bson.ObjectID{}, app.Invalid{Field: "customer_id", Message: "ID khách hàng không hợp lệ"}
	}
	return id, nil
}

func (r *CustomerRepository) UpdateProfile(ctx context.Context, userID string, input app.ProfileInput) (app.Profile, error) {
	id, err := customerObjectID(userID)
	if err != nil {
		return app.Profile{}, err
	}
	result, err := r.customers.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"full_name": input.FullName, "phone": input.Phone, "updated_at": time.Now().UTC()}, "$inc": bson.M{"profile_version": 1}})
	if err != nil {
		return app.Profile{}, fmt.Errorf("update customer profile: %w", err)
	}
	if result.MatchedCount == 0 {
		return app.Profile{}, app.ErrNotFound
	}
	return r.Profile(ctx, userID)
}

func (r *CustomerRepository) ReplaceAddresses(ctx context.Context, userID string, addresses []app.Address, expectedVersion int64) (app.Profile, error) {
	id, err := customerObjectID(userID)
	if err != nil {
		return app.Profile{}, err
	}
	if addresses == nil {
		addresses = []app.Address{}
	}
	result, err := r.customers.UpdateOne(ctx, customerVersionFilter(id, expectedVersion), bson.M{"$set": bson.M{"addresses": addresses, "updated_at": time.Now().UTC()}, "$inc": bson.M{"profile_version": 1}})
	if err != nil {
		return app.Profile{}, fmt.Errorf("update customer addresses: %w", err)
	}
	if result.MatchedCount == 0 {
		if exists, _ := r.customerExists(ctx, id); exists {
			return app.Profile{}, app.ErrConflict
		}
		return app.Profile{}, app.ErrNotFound
	}
	return r.Profile(ctx, userID)
}

func (r *CustomerRepository) ReplaceMeasurements(ctx context.Context, userID string, measurements *app.Measurements, notes string, expectedVersion int64) (app.Profile, error) {
	id, err := customerObjectID(userID)
	if err != nil {
		return app.Profile{}, err
	}
	update := bson.M{"$set": bson.M{"updated_at": time.Now().UTC()}}
	set := update["$set"].(bson.M)
	if measurements == nil {
		update["$unset"] = bson.M{"measurement_profile": "", "measurement_notes": ""}
	} else {
		set["measurement_profile"] = measurements
		if notes == "" {
			update["$unset"] = bson.M{"measurement_notes": ""}
		} else {
			set["measurement_notes"] = notes
		}
	}
	update["$inc"] = bson.M{"profile_version": 1}
	result, err := r.customers.UpdateOne(ctx, customerVersionFilter(id, expectedVersion), update)
	if err != nil {
		return app.Profile{}, fmt.Errorf("update customer measurements: %w", err)
	}
	if result.MatchedCount == 0 {
		if exists, _ := r.customerExists(ctx, id); exists {
			return app.Profile{}, app.ErrConflict
		}
		return app.Profile{}, app.ErrNotFound
	}
	return r.Profile(ctx, userID)
}

func customerVersionFilter(id bson.ObjectID, version int64) bson.M {
	return bson.M{"_id": id, "$or": bson.A{bson.M{"profile_version": version}, bson.M{"profile_version": bson.M{"$exists": false}}}}
}

func (r *CustomerRepository) customerExists(ctx context.Context, id bson.ObjectID) (bool, error) {
	count, err := r.customers.CountDocuments(ctx, bson.M{"_id": id})
	return count > 0, err
}

func (r *CustomerRepository) Wishlist(ctx context.Context, userID string) ([]app.WishlistItem, error) {
	id, err := customerObjectID(userID)
	if err != nil {
		return nil, err
	}
	var doc customerProfileDocument
	if err = r.customers.FindOne(ctx, bson.M{"_id": id}).Decode(&doc); errors.Is(err, driver.ErrNoDocuments) {
		return nil, app.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("read customer wishlist: %w", err)
	}
	visible, err := r.visibleWishlistProducts(ctx, doc.WishlistIDs)
	if err != nil {
		return nil, err
	}
	items := make([]app.WishlistItem, 0, len(doc.WishlistIDs))
	for _, productID := range doc.WishlistIDs {
		item := app.WishlistItem{ID: productID.Hex()}
		if product, ok := visible[productID.Hex()]; ok {
			productCopy := product
			item.Product = &productCopy
		} else {
			item.Product = &app.WishlistProduct{ID: productID.Hex(), Unavailable: true}
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *CustomerRepository) visibleWishlistProducts(ctx context.Context, ids []bson.ObjectID) (map[string]app.WishlistProduct, error) {
	products := make(map[string]app.WishlistProduct)
	if len(ids) == 0 {
		return products, nil
	}
	pipeline := bson.A{
		bson.M{"$match": bson.M{"_id": bson.M{"$in": ids}, "is_active": true}},
		bson.M{"$lookup": bson.M{"from": "categories", "localField": "category_id", "foreignField": "_id", "as": "_category"}},
		bson.M{"$unwind": "$_category"},
		bson.M{"$graphLookup": bson.M{"from": "categories", "startWith": "$_category.parent_id", "connectFromField": "parent_id", "connectToField": "_id", "as": "_ancestors"}},
		bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
			bson.M{"$eq": bson.A{"$_category.is_active", true}},
			bson.M{"$allElementsTrue": bson.A{bson.M{"$map": bson.M{
				"input": "$_ancestors", "as": "ancestor", "in": bson.M{"$eq": bson.A{"$$ancestor.is_active", true}},
			}}}},
			bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$_category.parent_id", nil}}, nil}},
				bson.M{"$gt": bson.A{
					bson.M{"$size": bson.M{"$filter": bson.M{
						"input": "$_ancestors", "as": "ancestor", "cond": bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$$ancestor.parent_id", nil}}, nil}},
					}}},
					0,
				}},
			}},
		}}}},
		bson.M{"$project": bson.M{"_id": 1, "slug": 1, "name": 1, "description": 1, "material": 1, "price": 1, "image": 1, "stock": 1, "bespoke": 1}},
	}
	cursor, err := r.products.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("read visible wishlist products: %w", err)
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode wishlist product: %w", err)
		}
		id := asID(raw["_id"])
		imageMap := asMap(raw["image"])
		var image *app.Image
		if src := asString(imageMap["src"]); src != "" {
			image = &app.Image{Src: src, Alt: asString(imageMap["alt"])}
		}
		stock := asInt64(raw["stock"])
		products[id] = app.WishlistProduct{ID: id, Slug: asString(raw["slug"]), Name: asString(raw["name"]), Description: asString(raw["description"]), Material: asString(raw["material"]), Price: asInt64(raw["price"]), Image: image, Stock: stock, ReadyAvailable: stock > 0, Bespoke: asBool(raw["bespoke"])}
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate wishlist products: %w", err)
	}
	return products, nil
}

func (r *CustomerRepository) AddWishlist(ctx context.Context, userID, productID string) error {
	customerID, err := customerObjectID(userID)
	if err != nil {
		return err
	}
	productOID, err := bson.ObjectIDFromHex(productID)
	visible, visibleErr := r.visibleWishlistProducts(ctx, []bson.ObjectID{productOID})
	if err != nil || visibleErr != nil || len(visible) == 0 {
		return app.ErrProductNotFound
	}
	result, err := r.customers.UpdateOne(ctx, bson.M{"_id": customerID}, bson.M{"$addToSet": bson.M{"wishlist_product_ids": productOID}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("add wishlist item: %w", err)
	}
	if result.MatchedCount == 0 {
		return app.ErrNotFound
	}
	return nil
}

func (r *CustomerRepository) RemoveWishlist(ctx context.Context, userID, productID string) error {
	customerID, err := customerObjectID(userID)
	if err != nil {
		return err
	}
	productOID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return app.ErrProductNotFound
	}
	result, err := r.customers.UpdateOne(ctx, bson.M{"_id": customerID}, bson.M{"$pull": bson.M{"wishlist_product_ids": productOID}, "$set": bson.M{"updated_at": time.Now().UTC()}})
	if err != nil {
		return fmt.Errorf("remove wishlist item: %w", err)
	}
	if result.MatchedCount == 0 {
		return app.ErrNotFound
	}
	return nil
}
