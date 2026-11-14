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

type AdminCatalogRepository struct{ categories, products, state *driver.Collection }

func NewAdminCatalogRepository(db *driver.Database) *AdminCatalogRepository {
	return &AdminCatalogRepository{categories: db.Collection("categories"), products: db.Collection("products"), state: db.Collection("catalog_admin_state")}
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
		if id, ok := v["_id"]; ok {
			out["id"] = adminValue(id)
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
		if id, ok := v["_id"]; ok {
			out["id"] = adminValue(id)
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
		for _, entry := range v {
			if entry.Key == "_id" {
				out["id"] = adminValue(entry.Value)
			}
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
	unset := bson.M{"old_price": ""} // Remove legacy fallback whenever comparison pricing is saved.
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
	set["version"] = int64(1)
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
		id, err := oid(parent)
		if err != nil {
			return app.Invalid{Field: "parent_id", Message: "ID danh mục cha không hợp lệ"}
		}
		// ObjectIDs are case-insensitive hexadecimal values, not raw strings.
		canonical := id.Hex()
		if seen[canonical] {
			return app.Invalid{Field: "parent_id", Message: "Quan hệ cha tạo thành vòng lặp"}
		}
		seen[canonical] = true
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

// A metadata version is optimistic concurrency control for full-form PUTs.
// Legacy documents without a version start at zero; the first successful
// update stores version=1. Variant/inventory operations don't bump this
// version because they are excluded from the metadata replacement payload.
func metadataVersionFilter(expected int64) bson.M {
	if expected == 0 {
		return bson.M{"$or": bson.A{bson.M{"version": bson.M{"$exists": false}}, bson.M{"version": int64(0)}}}
	}
	return bson.M{"version": expected}
}
func withMetadataVersion(id bson.ObjectID, expected int64) bson.M {
	filter := bson.M{"_id": id}
	for k, v := range metadataVersionFilter(expected) {
		filter[k] = v
	}
	return filter
}
func (r *AdminCatalogRepository) missingOrConflict(ctx context.Context, collection *driver.Collection, id bson.ObjectID) error {
	err := collection.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"_id": 1})).Err()
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.ErrNotFound
	}
	if err != nil {
		return err
	}
	return app.ErrConflict
}
func (r *AdminCatalogRepository) updateCategoryRecord(ctx context.Context, oidValue bson.ObjectID, v app.CategoryInput) (map[string]any, error) {
	set, unset, err := categoryFields(v)
	if err != nil {
		return nil, err
	}
	update := bson.M{"$set": set, "$inc": bson.M{"version": int64(1)}}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result := r.categories.FindOneAndUpdate(ctx, withMetadataVersion(oidValue, *v.ExpectedVersion), update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, r.missingOrConflict(ctx, r.categories, oidValue)
		}
		return nil, mongoWriteErr(err)
	}
	return adminDocument(doc), nil
}
func (r *AdminCatalogRepository) reparentCategoryTransaction(ctx context.Context, oidValue bson.ObjectID, v app.CategoryInput) (map[string]any, error) {
	// A shared document is deliberately written before checking ancestry.
	// Conflicting transactions cannot both validate against the old tree and
	// commit a cycle. This requires a replica set; no unsafe standalone
	// read-then-write fallback is permitted for hierarchy changes.
	session, err := r.categories.Database().Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(context.Background())
	value, err := session.WithTransaction(ctx, func(tx context.Context) (any, error) {
		_, lockErr := r.state.UpdateOne(tx, bson.M{"_id": "category-tree"}, bson.M{"$inc": bson.M{"revision": int64(1)}}, options.UpdateOne().SetUpsert(true))
		if lockErr != nil {
			return nil, lockErr
		}
		if err := r.verifyParent(tx, oidValue, v.ParentID); err != nil {
			return nil, err
		}
		return r.updateCategoryRecord(tx, oidValue, v)
	})
	if err != nil {
		var cmdErr driver.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 20 {
			return nil, app.ErrTransactionsRequired
		}
		return nil, err
	}
	return value.(map[string]any), nil
}
func (r *AdminCatalogRepository) UpdateCategory(ctx context.Context, id string, v app.CategoryInput) (map[string]any, error) {
	oidValue, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	var current bson.M
	if err := r.categories.FindOne(ctx, bson.M{"_id": oidValue}, options.FindOne().SetProjection(bson.M{"parent_id": 1})).Decode(&current); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, app.ErrNotFound
		}
		return nil, err
	}
	parent, err := optionalParent(v.ParentID)
	if err != nil {
		return nil, app.Invalid{Field: "parent_id", Message: "Mã danh mục cha không hợp lệ"}
	}
	if parent == current["parent_id"] || (parent == nil && current["parent_id"] == nil) {
		return r.updateCategoryRecord(ctx, oidValue, v)
	}
	return r.reparentCategoryTransaction(ctx, oidValue, v)
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
	set["version"] = int64(1)
	set["variants"] = v.Variants
	var total int64
	for _, item := range v.Variants {
		total += item.Stock
	}
	set["stock"] = total
	set["sizes"] = sizesFromVariants(v.Variants)
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
	update := bson.M{"$set": set, "$inc": bson.M{"version": int64(1)}}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	result := r.products.FindOneAndUpdate(ctx, withMetadataVersion(itemID, *v.ExpectedVersion), update, options.FindOneAndUpdate().SetReturnDocument(options.After))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, r.missingOrConflict(ctx, r.products, itemID)
		}
		return nil, mongoWriteErr(err)
	}
	return adminDocument(doc), nil
}

// Product display sizes are materialized from the authoritative SKUs in the
// same atomic update. A pipeline reads the current array, rather than writing
// a stale client copy of stock or replacing variants after a concurrent order.
func variantSizesExpression() bson.M {
	return bson.M{"$setUnion": bson.A{bson.M{"$map": bson.M{"input": "$variants", "as": "variant", "in": "$$variant.size"}}, bson.A{}}}
}
func (r *AdminCatalogRepository) AddVariant(ctx context.Context, id string, v app.Variant) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	// A legacy product with positive aggregate stock but zero SKUs cannot
	// silently acquire a zero-stock first SKU. It needs a Q-07 reconciliation.
	filter := bson.M{"_id": itemID,
		"$nor": bson.A{bson.M{"variants.sku": v.SKU}, bson.M{"variants": bson.M{"$elemMatch": bson.M{"color": v.Color, "size": v.Size}}}},
		"$expr": bson.M{"$eq": bson.A{
			bson.M{"$ifNull": bson.A{"$stock", 0}},
			bson.M{"$sum": bson.M{"$ifNull": bson.A{"$variants.stock", bson.A{}}}},
		}},
	}
	pipeline := driver.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			"variants":   bson.M{"$concatArrays": bson.A{bson.M{"$ifNull": bson.A{"$variants", bson.A{}}}, bson.M{"$literal": bson.A{v}}}},
			"updated_at": time.Now().UTC(),
		}}},
		bson.D{{Key: "$set", Value: bson.M{"sizes": variantSizesExpression()}}},
	}
	result := r.products.FindOneAndUpdate(ctx, filter, pipeline, options.FindOneAndUpdate().SetReturnDocument(options.After).SetCollation(&options.Collation{Locale: "vi", Strength: 2}))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, r.missingOrConflict(ctx, r.products, itemID)
		}
		return nil, mongoWriteErr(err)
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
	filter := bson.M{"_id": itemID, "variants.sku": sku,
		"$nor": bson.A{bson.M{"variants": bson.M{"$elemMatch": bson.M{"sku": bson.M{"$ne": sku}, "color": v.Color, "size": v.Size}}}},
	}
	// MongoDB evaluates both stages atomically on the latest product document,
	// preserving SKU stock even if an inventory adjustment runs concurrently.
	pipeline := driver.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{"variants": bson.M{"$map": bson.M{
			"input": "$variants", "as": "variant", "in": bson.M{"$cond": bson.A{
				bson.M{"$eq": bson.A{"$$variant.sku", sku}},
				bson.M{"$mergeObjects": bson.A{"$$variant", bson.M{"$literal": bson.M{"color": v.Color, "size": v.Size}}}},
				"$$variant",
			}},
		}}, "updated_at": time.Now().UTC()}}},
		bson.D{{Key: "$set", Value: bson.M{"sizes": variantSizesExpression()}}},
	}
	result := r.products.FindOneAndUpdate(ctx, filter, pipeline, options.FindOneAndUpdate().SetReturnDocument(options.After).SetCollation(&options.Collation{Locale: "vi", Strength: 2}))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, r.missingVariantOrConflict(ctx, itemID, sku)
		}
		return nil, mongoWriteErr(err)
	}
	return adminDocument(doc), nil
}
func (r *AdminCatalogRepository) missingVariantOrConflict(ctx context.Context, id bson.ObjectID, sku string) error {
	var doc bson.M
	err := r.products.FindOne(ctx, bson.M{"_id": id}, options.FindOne().SetProjection(bson.M{"variants.sku": 1})).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.ErrNotFound
	}
	if err != nil {
		return err
	}
	for _, entry := range asSlice(doc["variants"]) {
		if asString(asMap(entry)["sku"]) == sku {
			return app.ErrConflict
		}
	}
	return app.ErrNotFound
}

// Each request id is stored in the same product document and atomic mutation.
// The store of processed IDs is intentionally bounded; exhaustion FAILS
// CLOSED rather than silently forgetting keys and applying a retry twice.
const inventoryRequestLimit = 10000

func inventoryMutation(id bson.ObjectID, sku string, delta int64, requestID string) (bson.M, bson.M) {
	variant := bson.M{"sku": sku}
	filter := bson.M{"_id": id,
		"variants":                        bson.M{"$elemMatch": variant},
		"inventory_operations.request_id": bson.M{"$ne": requestID},
		"$expr":                           bson.M{"$lt": bson.A{bson.M{"$size": bson.M{"$ifNull": bson.A{"$inventory_operations", bson.A{}}}}, inventoryRequestLimit}},
	}
	if delta < 0 {
		needed := -delta
		variant["stock"] = bson.M{"$gte": needed}
		filter["stock"] = bson.M{"$gte": needed}
	}
	update := bson.M{
		"$inc":  bson.M{"stock": delta, "variants.$[target].stock": delta},
		"$set":  bson.M{"updated_at": time.Now().UTC()},
		"$push": bson.M{"inventory_operations": bson.M{"request_id": requestID, "sku": sku, "delta": delta, "at": time.Now().UTC()}},
	}
	return filter, update
}

// Filtered positional updates avoid ambiguity with the separate
// inventory_operations array used for retry deduplication.
func inventoryArrayFilters(sku string, delta int64) []any {
	filter := bson.M{"target.sku": sku}
	if delta < 0 {
		filter["target.stock"] = bson.M{"$gte": -delta}
	}
	return []any{filter}
}
func (r *AdminCatalogRepository) AdjustInventory(ctx context.Context, id, sku string, delta int64, requestID string) (map[string]any, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	filter, update := inventoryMutation(itemID, sku, delta, requestID)
	result := r.products.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After).SetArrayFilters(inventoryArrayFilters(sku, delta)))
	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		if !errors.Is(err, driver.ErrNoDocuments) {
			return nil, err
		}
		// The write may have committed but the response was lost. Return the
		// current product for a replay of that exact request id + payload.
		if err := r.products.FindOne(ctx, bson.M{"_id": itemID}).Decode(&doc); err != nil {
			if errors.Is(err, driver.ErrNoDocuments) {
				return nil, app.ErrNotFound
			}
			return nil, err
		}
		for _, entry := range asSlice(doc["inventory_operations"]) {
			op := asMap(entry)
			if asString(op["request_id"]) == requestID {
				if asString(op["sku"]) == sku && asInt64(op["delta"]) == delta {
					return adminDocument(doc), nil
				}
				return nil, app.ErrConflict // same key, different operation
			}
		}
		for _, entry := range asSlice(doc["variants"]) {
			if asString(asMap(entry)["sku"]) == sku {
				return nil, app.ErrConflict
			}
		}
		return nil, app.ErrNotFound
	}
	return adminDocument(doc), nil
}

func sizesFromVariants(variants []app.Variant) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(variants))
	for _, variant := range variants {
		if variant.Size != "" && !seen[variant.Size] {
			seen[variant.Size] = true
			out = append(out, variant.Size)
		}
	}
	return out
}
