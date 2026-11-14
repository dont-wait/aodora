package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	model "github.com/your-org/aodora/server/internal/domain/catalog"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// CatalogRepository implements the read-only MongoDB catalog.
type CatalogRepository struct {
	categories *driver.Collection
	products   *driver.Collection
	reviews    *driver.Collection
}

func NewCatalogRepository(db *driver.Database) *CatalogRepository {
	return &CatalogRepository{
		categories: db.Collection("categories"),
		products:   db.Collection("products"),
		reviews:    db.Collection("reviews"),
	}
}

// Keep MongoDB work below the HTTP response budget even if the caller supplies
// a context without a deadline. Parent request cancellation still takes priority.
const catalogQueryTimeout = 8 * time.Second

func catalogContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, catalogQueryTimeout)
}

// Public navigation applies the same active-ancestor/root rule as product
// listing, so an inactive parent cannot leak active child categories into tabs.
func publicCategoriesPipeline() bson.A {
	return bson.A{
		bson.M{"$match": bson.M{"is_active": true}},
		bson.M{"$graphLookup": bson.M{
			"from": "categories", "startWith": "$parent_id",
			"connectFromField": "parent_id", "connectToField": "_id", "as": "_ancestors",
		}},
		bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
			bson.M{"$allElementsTrue": bson.A{bson.M{"$map": bson.M{
				"input": "$_ancestors", "as": "ancestor",
				"in": bson.M{"$eq": bson.A{"$$ancestor.is_active", true}},
			}}}},
			bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$parent_id", nil}}, nil}},
				bson.M{"$gt": bson.A{
					bson.M{"$size": bson.M{"$filter": bson.M{
						"input": "$_ancestors", "as": "ancestor",
						"cond": bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$$ancestor.parent_id", nil}}, nil}},
					}}}, 0,
				}},
			}},
		}}}},
		bson.M{"$sort": bson.D{{Key: "sort_order", Value: 1}, {Key: "name", Value: 1}, {Key: "_id", Value: 1}}},
		bson.M{"$project": bson.M{"_id": 1, "id": 1, "slug": 1, "name": 1, "description": 1,
			"parent_id": 1, "is_active": 1, "sort_order": 1, "image": 1, "badge": 1}},
	}
}
func (r *CatalogRepository) Categories(ctx context.Context) ([]model.Category, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	cursor, err := r.categories.Aggregate(ctx, publicCategoriesPipeline(), options.Aggregate().SetCollation(&options.Collation{Locale: "vi", Strength: 2}))
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer cursor.Close(ctx)
	result := make([]model.Category, 0)
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, fmt.Errorf("decode category: %w", err)
		}
		result = append(result, categoryFromBSON(raw))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return result, nil
}

// categoryVisibilityStages validates the category and its complete parent chain
// while reading products. A missing, disabled, cyclic or orphan parent hides the
// product. This also closes the stale service-side category check window.
func categoryVisibilityStages() bson.A {
	return bson.A{
		bson.M{"$lookup": bson.M{
			"from": "categories", "localField": "category_id", "foreignField": "_id", "as": "_category",
		}},
		bson.M{"$unwind": "$_category"},
		bson.M{"$graphLookup": bson.M{
			"from": "categories", "startWith": "$_category.parent_id",
			"connectFromField": "parent_id", "connectToField": "_id", "as": "_ancestors",
		}},
		bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
			bson.M{"$eq": bson.A{"$_category.is_active", true}},
			bson.M{"$allElementsTrue": bson.A{bson.M{"$map": bson.M{
				"input": "$_ancestors", "as": "ancestor",
				"in": bson.M{"$eq": bson.A{"$$ancestor.is_active", true}},
			}}}},
			// An ancestor chain must terminate at a root (null/missing
			// parent_id), rather than at a deleted category or in a cycle.
			bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$_category.parent_id", nil}}, nil}},
				bson.M{"$gt": bson.A{
					bson.M{"$size": bson.M{"$filter": bson.M{
						"input": "$_ancestors", "as": "ancestor",
						"cond": bson.M{"$eq": bson.A{
							bson.M{"$ifNull": bson.A{"$$ancestor.parent_id", nil}}, nil,
						}},
					}}},
					0,
				}},
			}},
		}}}},
	}
}

// productPagePipeline reads data and pagination metadata in one aggregate.
// The equality/filter and indexed sort precede the category join; the facet
// counts only visible products and applies the offset to the same result set.
func productPagePipeline(filter model.Filter, activeCategoryIDs []string) bson.A {
	pipeline := bson.A{
		bson.M{"$match": productFilter(filter, activeCategoryIDs)},
		bson.M{"$sort": productSort(filter.Sort)},
	}
	pipeline = append(pipeline, categoryVisibilityStages()...)
	pipeline = append(pipeline, bson.M{"$facet": bson.M{
		"metadata": bson.A{bson.M{"$count": "total"}},
		"data": bson.A{
			bson.M{"$skip": (filter.Page - 1) * filter.PageSize},
			bson.M{"$limit": filter.PageSize},
			bson.M{"$project": publicProductSummaryProjection()},
		},
	}})
	return pipeline
}

func (r *CatalogRepository) ListProducts(ctx context.Context, filter model.Filter, activeCategoryIDs []string) ([]model.Product, int64, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	aggregateOptions := options.Aggregate()
	// Keep text-index searches on their binary collation. For ordinary name
	// browsing, use the Vietnamese collation (and matching indexes) instead
	// of comparing UTF-8 bytes, which puts accented names out of order.
	if usesVietnameseNameCollation(filter) {
		aggregateOptions.SetCollation(&options.Collation{Locale: "vi", Strength: 2})
	}
	cursor, err := r.products.Aggregate(ctx, productPagePipeline(filter, activeCategoryIDs), aggregateOptions)
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog products: %w", err)
	}
	defer cursor.Close(ctx)
	if !cursor.Next(ctx) {
		if err := cursor.Err(); err != nil {
			return nil, 0, fmt.Errorf("read catalog page: %w", err)
		}
		return []model.Product{}, 0, nil
	}
	var result bson.M
	if err := cursor.Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode catalog page: %w", err)
	}
	total := int64(0)
	metadata := asSlice(result["metadata"])
	if len(metadata) > 0 {
		total = asInt64(asMap(metadata[0])["total"])
	}
	products := make([]model.Product, 0)
	for _, row := range asSlice(result["data"]) {
		product, err := productFromBSON(asMap(row))
		if err != nil {
			return nil, 0, err
		}
		products = append(products, product)
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate catalog page: %w", err)
	}
	return products, total, nil
}

func productDetailPipeline(slug string) bson.A {
	pipeline := bson.A{bson.M{"$match": bson.M{"slug": slug, "is_active": true}}}
	pipeline = append(pipeline, categoryVisibilityStages()...)
	return append(pipeline,
		bson.M{"$limit": 1},
		bson.M{"$project": publicProductProjection()},
	)
}

func (r *CatalogRepository) ProductBySlug(ctx context.Context, slug string) (*model.Product, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	cursor, err := r.products.Aggregate(ctx, productDetailPipeline(slug))
	if err != nil {
		return nil, fmt.Errorf("find catalog product by slug: %w", err)
	}
	defer cursor.Close(ctx)
	if !cursor.Next(ctx) {
		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("read product detail: %w", err)
		}
		return nil, nil
	}
	var raw bson.M
	if err := cursor.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode product detail: %w", err)
	}
	product, err := productFromBSON(raw)
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *CatalogRepository) ReviewSummary(ctx context.Context, productID string) (model.ReviewSummary, error) {
	ctx, cancel := catalogContext(ctx)
	defer cancel()
	id, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return model.ReviewSummary{}, fmt.Errorf("invalid product id in review summary: %w", err)
	}
	pipeline := bson.A{
		bson.M{"$match": bson.M{"product_id": id}},
		bson.M{"$group": bson.M{"_id": nil, "count": bson.M{"$sum": 1}, "average": bson.M{"$avg": "$rating"}}},
	}
	cursor, err := r.reviews.Aggregate(ctx, pipeline)
	if err != nil {
		return model.ReviewSummary{}, fmt.Errorf("aggregate product reviews: %w", err)
	}
	defer cursor.Close(ctx)
	if !cursor.Next(ctx) {
		if err := cursor.Err(); err != nil {
			return model.ReviewSummary{}, fmt.Errorf("iterate review stats: %w", err)
		}
		return model.ReviewSummary{}, nil
	}
	var raw bson.M
	if err := cursor.Decode(&raw); err != nil {
		return model.ReviewSummary{}, fmt.Errorf("decode review stats: %w", err)
	}
	summary := model.ReviewSummary{Count: asInt64(raw["count"]), Average: asFloat64(raw["average"])}
	// Avoid returning unrounded floating representation to the client.
	summary.Average = math.Round(summary.Average*100) / 100
	return summary, nil
}

// Phrase matching avoids the default OR semantics of MongoDB $text: a
// multi-word product name must not match unrelated products just because
// they share a generic word (for example, "Áo"). Strip literal quotation
// marks to keep the entire user-entered text inside one phrase, not as
// additional MongoDB text query operators.
func catalogSearchPhrase(search string) string {
	cleaned := strings.Join(strings.Fields(strings.ReplaceAll(search, `"`, "")), " ")
	return `"` + cleaned + `"`
}

// MongoDB text indexes are binary-collated; the Vietnamese collation is used
// for name sorting only when a text-index query is not in progress.
func usesVietnameseNameCollation(f model.Filter) bool {
	return strings.TrimSpace(f.Search) == "" && (f.Sort == "name_asc" || f.Sort == "name_desc")
}

func productFilter(f model.Filter, categoryIDs []string) bson.M {
	ids := make(bson.A, 0, len(categoryIDs))
	for _, value := range categoryIDs {
		if id, err := bson.ObjectIDFromHex(value); err == nil {
			ids = append(ids, id)
		}
	}
	terms := bson.A{bson.M{"is_active": true}, bson.M{"category_id": bson.M{"$in": ids}}}
	if f.Search != "" {
		// Use the existing text index on name + description rather than a
		// collection-scanning unanchored case-insensitive regex.
		terms = append(terms, bson.M{"$text": bson.M{"$search": catalogSearchPhrase(f.Search)}})
	}
	price := bson.M{}
	if f.MinPrice != nil {
		price["$gte"] = *f.MinPrice
	}
	if f.MaxPrice != nil {
		price["$lte"] = *f.MaxPrice
	}
	if len(price) != 0 {
		terms = append(terms, bson.M{"price": price})
	}
	if f.Material != "" {
		terms = append(terms, bson.M{"material": bson.Regex{
			Pattern: "^" + regexp.QuoteMeta(f.Material) + "$", Options: "i",
		}})
	}
	if f.Color != "" || f.Size != "" {
		variant := bson.M{}
		fallback := bson.A{bson.M{"$or": bson.A{
			bson.M{"variants": bson.M{"$exists": false}},
			bson.M{"variants": bson.M{"$size": 0}},
		}}}
		if f.Color != "" {
			exact := bson.Regex{Pattern: "^" + regexp.QuoteMeta(f.Color) + "$", Options: "i"}
			variant["color"] = exact
			fallback = append(fallback, bson.M{"$or": bson.A{
				bson.M{"colors": bson.M{"$elemMatch": bson.M{"name": exact}}},
				bson.M{"colors": bson.M{"$elemMatch": bson.M{"hex": exact}}},
			}})
		}
		if f.Size != "" {
			exact := bson.Regex{Pattern: "^" + regexp.QuoteMeta(f.Size) + "$", Options: "i"}
			variant["size"] = exact
			fallback = append(fallback, bson.M{"sizes": exact})
		}
		// An existing SKU must satisfy BOTH requested attributes in ONE
		// variant. Display-only colors/sizes may match only without variants.
		terms = append(terms, bson.M{"$or": bson.A{
			bson.M{"variants": bson.M{"$elemMatch": variant}},
			bson.M{"$and": fallback},
		}})
	}
	return bson.M{"$and": terms}
}

func productSort(sortKey string) bson.D {
	switch sortKey {
	case "price_asc", "price-asc":
		return bson.D{{Key: "price", Value: 1}, {Key: "_id", Value: 1}}
	case "price_desc", "price-desc":
		return bson.D{{Key: "price", Value: -1}, {Key: "_id", Value: -1}}
	case "name_asc":
		return bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}}
	case "name_desc":
		return bson.D{{Key: "name", Value: -1}, {Key: "_id", Value: -1}}
	default:
		return bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}
	}
}

// Read only the card fields in list requests; the large story/fabric/craft
// payload is loaded for a single product on the detail endpoint.
func publicProductSummaryProjection() bson.M {
	return bson.M{
		"_id": 1, "category_id": 1, "slug": 1, "name": 1, "description": 1,
		"material": 1, "price": 1, "old_price": 1, "compare_at_price": 1,
		"stock": 1, "sku": 1, "variants": 1, "images": 1, "image": 1,
		"badge": 1, "tag": 1, "colors": 1, "sizes": 1,
		"bespoke": 1, "created_at": 1,
	}
}

// A strict root-field allowlist prevents accidental exposure if documents gain
// private/admin-only fields in future migrations.
func publicProductProjection() bson.M {
	return bson.M{
		"_id": 1, "category_id": 1, "slug": 1, "name": 1, "description": 1,
		"material": 1, "price": 1, "old_price": 1, "compare_at_price": 1,
		"stock": 1, "sku": 1, "variants": 1, "images": 1, "image": 1,
		"gallery": 1, "badge": 1, "tag": 1, "colors": 1, "color_note": 1,
		"sizes": 1, "bespoke": 1, "cta": 1, "bundle": 1,
		"collection": 1, "discount": 1, "availability": 1,
		"accessories": 1, "trust": 1, "story": 1, "fabric": 1,
		"craft": 1, "created_at": 1, "updated_at": 1,
	}
}

func categoryFromBSON(m bson.M) model.Category {
	return model.Category{
		ID: asID(m["_id"]), Key: asString(m["id"]),
		Slug: asString(m["slug"]), Name: asString(m["name"]),
		Description: asString(m["description"]), ParentID: asID(m["parent_id"]),
		IsActive: asBool(m["is_active"]), SortOrder: asInt64(m["sort_order"]),
		Image: readImage(m["image"]), Badge: readLabel(m["badge"]),
	}
}

func productFromBSON(raw bson.M) (model.Product, error) {
	price, err := asNumber(raw["price"])
	if err != nil {
		return model.Product{}, fmt.Errorf("product %s price: %w", asID(raw["_id"]), err)
	}
	p := model.Product{
		ID: asID(raw["_id"]), CategoryID: asID(raw["category_id"]),
		Slug: asString(raw["slug"]), Name: asString(raw["name"]),
		Description: asString(raw["description"]), Material: asString(raw["material"]),
		Price: price, Stock: asInt64(raw["stock"]), SKU: asString(raw["sku"]),
		Images: readStrings(raw["images"]), Image: readImage(raw["image"]),
		Gallery: readImages(raw["gallery"]), Badge: readLabel(raw["badge"]),
		Tag: readLabel(raw["tag"]), Colors: readColors(raw["colors"]),
		ColorNote: asString(raw["color_note"]), Sizes: readStrings(raw["sizes"]),
		Bespoke: asBool(raw["bespoke"]), CTA: asString(raw["cta"]),
		Bundle: asString(raw["bundle"]), Collection: asString(raw["collection"]),
		Discount: asInt64(raw["discount"]), Availability: asString(raw["availability"]),
		Trust: readTrust(raw["trust"]), Story: readStory(raw["story"]),
		Fabric: readFabric(raw["fabric"]), Craft: readCraft(raw["craft"]),
		CreatedAt: asTime(raw["created_at"]), UpdatedAt: asTime(raw["updated_at"]),
		Variants: []model.Variant{}, Accessories: []model.Accessory{},
	}
	if n := raw["old_price"]; n != nil {
		v, err := asNumber(n)
		if err != nil {
			return model.Product{}, fmt.Errorf("product %s old_price: %w", p.ID, err)
		}
		p.OldPrice = &v
	}
	if n := raw["compare_at_price"]; n != nil {
		v, err := asNumber(n)
		if err != nil {
			return model.Product{}, fmt.Errorf("product %s compare_at_price: %w", p.ID, err)
		}
		p.CompareAtPrice = &v
	}
	for _, value := range asSlice(raw["variants"]) {
		v := asMap(value)
		p.Variants = append(p.Variants, model.Variant{
			SKU: asString(v["sku"]), Color: asString(v["color"]),
			Size: asString(v["size"]), Stock: asInt64(v["stock"]),
		})
	}
	for _, value := range asSlice(raw["accessories"]) {
		a := asMap(value)
		amount, err := asNumber(a["price"])
		if err != nil {
			return model.Product{}, fmt.Errorf("product %s accessory price: %w", p.ID, err)
		}
		p.Accessories = append(p.Accessories, model.Accessory{
			ID: asString(a["id"]), Name: asString(a["name"]),
			Description: asString(a["description"]), Price: amount,
			Image: readImage(a["image"]), Tag: asString(a["tag"]),
			TagVariant: asString(a["tag_variant"]),
		})
	}
	if p.Image == nil && len(p.Images) > 0 {
		p.Image = &model.Image{Src: p.Images[0], Alt: p.Name}
	}
	if len(p.Images) == 0 && p.Image != nil {
		p.Images = []string{p.Image.Src}
	}
	if len(p.Colors) == 0 {
		seen := map[string]bool{}
		for _, v := range p.Variants {
			if v.Color != "" && !seen[v.Color] {
				p.Colors = append(p.Colors, model.Color{Name: v.Color})
				seen[v.Color] = true
			}
		}
	}
	if len(p.Sizes) == 0 {
		seen := map[string]bool{}
		for _, v := range p.Variants {
			if v.Size != "" && !seen[v.Size] {
				p.Sizes = append(p.Sizes, v.Size)
				seen[v.Size] = true
			}
		}
		sort.Strings(p.Sizes)
	}
	if p.Colors == nil {
		p.Colors = []model.Color{}
	}
	if p.Sizes == nil {
		p.Sizes = []string{}
	}
	if p.Gallery == nil {
		p.Gallery = []model.Image{}
	}
	if p.Images == nil {
		p.Images = []string{}
	}
	return p, nil
}

func readImage(value any) *model.Image {
	if value == nil {
		return nil
	}
	data := asMap(value)
	if len(data) == 0 {
		return nil
	}
	return &model.Image{Src: asString(data["src"]), Alt: asString(data["alt"])}
}

func readLabel(value any) *model.Label {
	if value == nil {
		return nil
	}
	data := asMap(value)
	if len(data) == 0 {
		return nil
	}
	return &model.Label{Label: asString(data["label"]), Variant: asString(data["variant"])}
}

func readStrings(value any) []string {
	result := make([]string, 0)
	for _, raw := range asSlice(value) {
		result = append(result, asString(raw))
	}
	return result
}

func readImages(value any) []model.Image {
	images := make([]model.Image, 0)
	for _, raw := range asSlice(value) {
		if img := readImage(raw); img != nil {
			images = append(images, *img)
		}
	}
	return images
}

func readColors(value any) []model.Color {
	result := make([]model.Color, 0)
	for _, item := range asSlice(value) {
		color := asMap(item)
		result = append(result, model.Color{Name: asString(color["name"]), Hex: asString(color["hex"])})
	}
	return result
}

func readTrust(value any) []model.TrustItem {
	result := make([]model.TrustItem, 0)
	for _, item := range asSlice(value) {
		v := asMap(item)
		result = append(result, model.TrustItem{
			Icon: asString(v["icon"]), Title: asString(v["title"]), Description: asString(v["description"]),
		})
	}
	return result
}

func readContent(value any) []model.ContentItem {
	result := make([]model.ContentItem, 0)
	for _, item := range asSlice(value) {
		v := asMap(item)
		result = append(result, model.ContentItem{
			Icon: asString(v["icon"]), Title: asString(v["title"]), Description: asString(v["description"]),
		})
	}
	return result
}

func readStory(value any) *model.Story {
	if value == nil {
		return nil
	}
	m := asMap(value)
	if len(m) == 0 {
		return nil
	}
	return &model.Story{
		Eyebrow: asString(m["eyebrow"]), Title: asString(m["title"]),
		Paragraphs: readStrings(m["paragraphs"]), Image: readImage(m["image"]),
	}
}

func readFabric(value any) *model.Fabric {
	if value == nil {
		return nil
	}
	m := asMap(value)
	if len(m) == 0 {
		return nil
	}
	return &model.Fabric{
		Eyebrow: asString(m["eyebrow"]), Title: asString(m["title"]),
		Description: asString(m["description"]), Image: readImage(m["image"]),
		Care: readContent(m["care"]),
	}
}

func readCraft(value any) *model.Craft {
	if value == nil {
		return nil
	}
	m := asMap(value)
	if len(m) == 0 {
		return nil
	}
	return &model.Craft{
		Eyebrow: asString(m["eyebrow"]), Title: asString(m["title"]),
		Description: asString(m["description"]), Image: readImage(m["image"]),
		Steps: readContent(m["steps"]),
	}
}

func asMap(value any) bson.M {
	switch v := value.(type) {
	case bson.M:
		return v
	case map[string]any:
		return bson.M(v)
	case bson.D:
		result := bson.M{}
		for _, entry := range v {
			result[entry.Key] = entry.Value
		}
		return result
	default:
		return bson.M{}
	}
}

func asSlice(value any) []any {
	switch v := value.(type) {
	case bson.A:
		return []any(v)
	case []any:
		return v
	default:
		return nil
	}
}

func asString(value any) string {
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}

func asID(value any) string {
	switch v := value.(type) {
	case bson.ObjectID:
		return v.Hex()
	case string:
		return v
	default:
		return ""
	}
}

func asBool(value any) bool { v, _ := value.(bool); return v }
func asInt64(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case bson.Decimal128:
		value, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return int64(value)
		}
	}
	return 0
}
func asFloat64(value any) float64 {
	switch v := value.(type) {
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case bson.Decimal128:
		result, _ := strconv.ParseFloat(v.String(), 64)
		return result
	}
	return 0
}
func asTime(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case bson.DateTime:
		return v.Time()
	}
	return time.Time{}
}
func asNumber(value any) (json.Number, error) {
	switch v := value.(type) {
	case int:
		return json.Number(strconv.Itoa(v)), nil
	case int32:
		return json.Number(strconv.FormatInt(int64(v), 10)), nil
	case int64:
		return json.Number(strconv.FormatInt(v, 10)), nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return "", fmt.Errorf("non-finite price")
		}
		return json.Number(strconv.FormatFloat(v, 'f', -1, 64)), nil
	case bson.Decimal128:
		value := v.String()
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) ||
			strings.EqualFold(value, "nan") {
			return "", fmt.Errorf("invalid decimal price %q", value)
		}
		return json.Number(value), nil
	case nil:
		return "", fmt.Errorf("missing price")
	default:
		return "", fmt.Errorf("unsupported price type %T", value)
	}
}
