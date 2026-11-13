package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	app "github.com/your-org/aodora/server/internal/application/catalogadmin"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type AdminCatalogRepository struct{ categories, products *driver.Collection }

func NewAdminCatalogRepository(db *driver.Database) *AdminCatalogRepository {
	return &AdminCatalogRepository{categories: db.Collection("categories"), products: db.Collection("products")}
}
func (r *AdminCatalogRepository) Categories(ctx context.Context) ([]map[string]any, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	cur, err := r.categories.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "sort_order", Value: 1}, {Key: "name", Value: 1}, {Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	return adminDecodeList(ctx, cur)
}
func (r *AdminCatalogRepository) Products(ctx context.Context, page, pageSize int64) (app.ProductPage, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	total, err := r.products.CountDocuments(ctx, bson.M{})
	if err != nil {
		return app.ProductPage{}, err
	}
	result := app.ProductPage{Items: []map[string]any{}, Total: total, Page: page, PageSize: pageSize}
	if total > 0 {
		result.TotalPages = (total + pageSize - 1) / pageSize
	}
	cur, err := r.products.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}, {Key: "_id", Value: -1}}).SetSkip((page-1)*pageSize).SetLimit(pageSize))
	if err != nil {
		return app.ProductPage{}, err
	}
	defer cur.Close(ctx)
	result.Items, err = adminDecodeList(ctx, cur)
	return result, err
}
func adminDecodeList(ctx context.Context, cur *driver.Cursor) ([]map[string]any, error) {
	result := make([]map[string]any, 0)
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		result = append(result, adminDocument(doc))
	}
	return result, cur.Err()
}
func adminValue(raw any) any {
	switch v := raw.(type) {
	case bson.ObjectID:
		return v.Hex()
	case bson.DateTime:
		return time.UnixMilli(int64(v)).UTC()
	case bson.M:
		out := map[string]any{}
		for k, x := range v {
			if k == "_id" {
				k = "id"
			}
			out[k] = adminValue(x)
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, x := range v {
			if k == "_id" {
				k = "id"
			}
			out[k] = adminValue(x)
		}
		return out
	case bson.D:
		out := map[string]any{}
		for _, entry := range v {
			k := entry.Key
			if k == "_id" {
				k = "id"
			}
			out[k] = adminValue(entry.Value)
		}
		return out
	case bson.A:
		out := make([]any, 0, len(v))
		for _, x := range v {
			out = append(out, adminValue(x))
		}
		return out
	case bson.Decimal128:
		n, err := asNumber(v)
		if err != nil {
			return nil
		}
		return n
	default:
		return raw
	}
}
func adminDocument(doc bson.M) map[string]any { return adminValue(doc).(map[string]any) }
func oid(value string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		return bson.ObjectID{}, app.Invalid{Field: "id", Message: "ID không hợp lệ"}
	}
	return id, nil
}
func optionalParent(value string) (any, error) {
	if value == "" {
		return nil, nil
	}
	id, err := oid(value)
	return id, err
}
func (r *AdminCatalogRepository) CategoryExists(ctx context.Context, value string) (bool, error) {
	id, err := oid(value)
	if err != nil {
		return false, nil
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	count, err := r.categories.CountDocuments(ctx, bson.M{"_id": id})
	return count > 0, err
}
func categoryFields(v app.CategoryInput) (bson.M, bson.M, error) {
	parent, err := optionalParent(v.ParentID)
	if err != nil {
		return nil, nil, app.Invalid{Field: "parent_id", Message: "Mã danh mục cha không hợp lệ"}
	}
	set := bson.M{"name": v.Name, "slug": v.Slug, "description": v.Description, "parent_id": parent, "sort_order": v.SortOrder, "is_active": v.IsActive, "updated_at": time.Now().UTC()}
	unset := bson.M{}
	if v.Image != nil {
		set["image"] = v.Image
	} else {
		unset["image"] = ""
	}
	return set, unset, nil
}
func productFields(v app.ProductInput) (bson.M, bson.M, error) {
	categoryID, err := oid(v.CategoryID)
	if err != nil {
		return nil, nil, app.Invalid{Field: "category_id", Message: "Mã danh mục không hợp lệ"}
	}
	now := time.Now().UTC()
	if v.Images == nil {
		v.Images = []string{}
	}
	if v.Gallery == nil {
		v.Gallery = []app.Image{}
	}
	if v.Colors == nil {
		v.Colors = []app.Color{}
	}
	if v.Accessories == nil {
		v.Accessories = []app.Accessory{}
	}
	set := bson.M{"name": v.Name, "slug": v.Slug, "category_id": categoryID, "description": v.Description, "material": v.Material, "price": v.Price, "images": v.Images, "gallery": v.Gallery, "colors": v.Colors, "accessories": v.Accessories, "is_active": v.IsActive, "updated_at": now}
	unset := bson.M{}
	if v.CompareAtPrice != nil {
		set["compare_at_price"] = *v.CompareAtPrice
	} else {
		unset["compare_at_price"] = ""
	}
	if v.Image != nil {
		set["image"] = v.Image
	} else {
		unset["image"] = ""
	}
	return set, unset, nil
}
func mongoWriteErr(err error) error {
	if err == nil {
		return nil
	}
	if driver.IsDuplicateKeyError(err) {
		return fmt.Errorf("%w: slug/SKU đã được sử dụng", app.ErrConflict)
	}
	return err
}
func (r *AdminCatalogRepository) CreateCategory(ctx context.Context, v app.CategoryInput) (map[string]any, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	set, _, err := categoryFields(v)
	if err != nil {
		return nil, err
	}
	if err := r.verifyParent(ctx, bson.ObjectID{}, v.ParentID); err != nil {
		return nil, err
	}
	set["_id"] = bson.NewObjectID()
	set["created_at"] = set["updated_at"]
	_, err = r.categories.InsertOne(ctx, set)
	if err != nil {
		return nil, mongoWriteErr(err)
	}
	return adminDocument(set), nil
}
func (r *AdminCatalogRepository) verifyParent(ctx context.Context, current bson.ObjectID, parent string) error {
	if parent == "" {
		return nil
	}
	seen := map[string]bool{}
	if current != (bson.ObjectID{}) {
		seen[current.Hex()] = true
	}
	for parent != "" {
		if seen[parent] {
			return app.Invalid{Field: "parent_id", Message: "Quan hệ cha tạo thành vòng lặp"}
		}
		seen[parent] = true
		id, err := oid(parent)
		if err != nil {
			return app.Invalid{Field: "parent_id", Message: "ID danh mục cha không hợp lệ"}
		}
		var row bson.M
		if err := r.categories.FindOne(ctx, bson.M{"_id": id}).Decode(&row); err != nil {
			if errors.Is(err, driver.ErrNoDocuments) {
				return app.Invalid{Field: "parent_id", Message: "Danh mục cha không tồn tại"}
			}
			return err
		}
		parent = asID(row["parent_id"])
	}
	return nil
}
func (r *AdminCatalogRepository) UpdateCategory(ctx context.Context, id string, v app.CategoryInput) (map[string]any, error) {
	oidValue, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	if err := r.verifyParent(ctx, oidValue, v.ParentID); err != nil {
		return nil, err
	}
	set, unset, err := categoryFields(v)
	if err != nil {
		return nil, err
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result := r.categories.FindOneAndUpdate(ctx, bson.M{"_id": oidValue}, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrNotFound
		}
		return nil, mongoWriteErr(err)
	}
	return adminDocument(doc), nil
}
func (r *AdminCatalogRepository) CreateProduct(ctx context.Context, v app.ProductCreate) (map[string]any, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	set, _, err := productFields(v.ProductInput)
	if err != nil {
		return nil, err
	}
	set["_id"] = bson.NewObjectID()
	set["created_at"] = set["updated_at"]
	set["variants"] = v.Variants
	var total int64
	for _, item := range v.Variants {
		total += item.Stock
	}
	set["stock"] = total
	_, err = r.products.InsertOne(ctx, set)
	if err != nil {
		return nil, mongoWriteErr(err)
	}
	return adminDocument(set), nil
}
func (r *AdminCatalogRepository) UpdateProduct(ctx context.Context, id string, v app.ProductInput) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	set, unset, err := productFields(v)
	if err != nil {
		return nil, err
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result := r.products.FindOneAndUpdate(ctx, bson.M{"_id": itemID}, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrNotFound
		}
		return nil, mongoWriteErr(err)
	}
	return adminDocument(doc), nil
}
func (r *AdminCatalogRepository) AddVariant(ctx context.Context, id string, v app.Variant) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	filter := bson.M{"_id": itemID, "$nor": bson.A{bson.M{"variants.sku": v.SKU}, bson.M{"variants": bson.M{"$elemMatch": bson.M{"color": v.Color, "size": v.Size}}}}}
	result := r.products.FindOneAndUpdate(ctx, filter, bson.M{"$push": bson.M{"variants": v}, "$set": bson.M{"updated_at": time.Now().UTC()}}, options.FindOneAndUpdate().SetReturnDocument(options.After).SetCollation(&options.Collation{Locale: "vi", Strength: 2}))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrConflict
		}
		return nil, err
	}
	return adminDocument(doc), nil
}
func (r *AdminCatalogRepository) UpdateVariant(ctx context.Context, id, sku string, v app.VariantDetails) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	filter := bson.M{"_id": itemID, "variants.sku": sku, "$nor": bson.A{bson.M{"variants": bson.M{"$elemMatch": bson.M{"sku": bson.M{"$ne": sku}, "color": v.Color, "size": v.Size}}}}}
	update := bson.M{"$set": bson.M{"variants.$.color": v.Color, "variants.$.size": v.Size, "updated_at": time.Now().UTC()}}
	result := r.products.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After).SetCollation(&options.Collation{Locale: "vi", Strength: 2}))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrConflict
		}
		return nil, err
	}
	return adminDocument(doc), nil
}

// AdjustInventory is a single-document conditional atomic update. DW-02 must
// use an equivalent SKU+aggregate guarded $inc when reserving/cancelling orders.
func inventoryMutation(id bson.ObjectID, sku string, delta int64) (bson.M, bson.M) {
	variant := bson.M{"sku": sku}
	filter := bson.M{"_id": id}
	if delta < 0 {
		needed := -delta
		variant["stock"] = bson.M{"$gte": needed}
		filter["stock"] = bson.M{"$gte": needed}
	}
	filter["variants"] = bson.M{"$elemMatch": variant}
	update := bson.M{"$inc": bson.M{"stock": delta, "variants.$.stock": delta}, "$set": bson.M{"updated_at": time.Now().UTC()}}
	return filter, update
}
func (r *AdminCatalogRepository) AdjustInventory(ctx context.Context, id, sku string, delta int64) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	filter, update := inventoryMutation(itemID, sku, delta)
	result := r.products.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrConflict
		}
		return nil, err
	}
	return adminDocument(doc), nil
}
