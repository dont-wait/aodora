package mongo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	app "github.com/your-org/aodora/server/internal/application/cart"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type CartRepository struct {
	carts      *driver.Collection
	products   *driver.Collection
	categories *driver.Collection
}

func NewCartRepository(db *driver.Database) *CartRepository {
	return &CartRepository{carts: db.Collection("carts"), products: db.Collection("products"), categories: db.Collection("categories")}
}

type storedCartItem struct {
	ProductID       bson.ObjectID `bson:"product_id"`
	SKU             string        `bson:"sku"`
	Quantity        int64         `bson:"quantity"`
	ColorName       string        `bson:"color_name,omitempty"`
	ColorHex        string        `bson:"color_hex,omitempty"`
	FulfillmentMode string        `bson:"fulfillment_mode"`
	UnitPrice       int64         `bson:"unit_price,omitempty"`
	AddedAt         time.Time     `bson:"added_at"`
}

type storedCart struct {
	ID                 bson.ObjectID    `bson:"_id"`
	CustomerID         bson.ObjectID    `bson:"customer_id"`
	Items              []storedCartItem `bson:"items"`
	LastMergeRequestID string           `bson:"last_merge_request_id,omitempty"`
	Version            int64            `bson:"version,omitempty"`
}

type visibleProduct struct {
	ID       string
	Slug     string
	Name     string
	SKU      string
	Image    string
	Price    int64
	Stock    int64
	Variants map[string]visibleVariant
}

type visibleVariant struct {
	Color string
	Size  string
	Stock int64
}

func (r *CartRepository) Cart(ctx context.Context, userID string) (app.Cart, error) {
	customerID, err := cartObjectID(userID)
	if err != nil {
		return app.Cart{}, err
	}
	var stored storedCart
	err = r.carts.FindOne(ctx, bson.M{"customer_id": customerID}).Decode(&stored)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.Cart{Items: []app.Item{}, Issues: []app.Item{}}, nil
	}
	if err != nil {
		return app.Cart{}, fmt.Errorf("read cart: %w", err)
	}
	return r.hydrate(ctx, stored.Items)
}

func (r *CartRepository) AddItem(ctx context.Context, userID string, selection app.Selection) (app.Cart, error) {
	customerID, err := cartObjectID(userID)
	if err != nil {
		return app.Cart{}, err
	}
	product, issue, validationErr := r.selectionProduct(ctx, selection)
	if validationErr != nil {
		return app.Cart{}, validationErr
	}
	if issue != "" {
		return app.Cart{}, app.ErrItemUnavailable
	}
	stored, err := r.loadOrCreate(ctx, customerID)
	if err != nil {
		return app.Cart{}, err
	}
	updated := mergeStoredItem(stored.Items, selection, product.Price)
	if err := r.saveItems(ctx, stored, updated, ""); err != nil {
		return app.Cart{}, err
	}
	return r.hydrate(ctx, updated)
}

func (r *CartRepository) UpdateItem(ctx context.Context, userID string, selection app.Selection) (app.Cart, error) {
	customerID, err := cartObjectID(userID)
	if err != nil {
		return app.Cart{}, err
	}
	product, issue, validationErr := r.selectionProduct(ctx, selection)
	if validationErr != nil {
		return app.Cart{}, validationErr
	}
	if issue != "" {
		return app.Cart{}, app.ErrItemUnavailable
	}
	var stored storedCart
	if err := r.carts.FindOne(ctx, bson.M{"customer_id": customerID}).Decode(&stored); errors.Is(err, driver.ErrNoDocuments) {
		return app.Cart{}, app.ErrItemUnavailable
	} else if err != nil {
		return app.Cart{}, fmt.Errorf("read cart: %w", err)
	}
	updated := replaceStoredItem(stored.Items, selection, product.Price)
	if len(updated) == len(stored.Items) && !containsStoredItem(stored.Items, selection) {
		return app.Cart{}, app.ErrItemUnavailable
	}
	if err := r.saveItems(ctx, stored, updated, ""); err != nil {
		return app.Cart{}, err
	}
	return r.hydrate(ctx, updated)
}

func (r *CartRepository) RemoveItem(ctx context.Context, userID, productID, sku string) (app.Cart, error) {
	customerID, err := cartObjectID(userID)
	if err != nil {
		return app.Cart{}, err
	}
	productOID, err := bson.ObjectIDFromHex(productID)
	if err != nil {
		return app.Cart{}, app.Invalid{Field: "product_id", Message: "ID sản phẩm không hợp lệ"}
	}
	var stored storedCart
	if err := r.carts.FindOne(ctx, bson.M{"customer_id": customerID}).Decode(&stored); errors.Is(err, driver.ErrNoDocuments) {
		return app.Cart{Items: []app.Item{}, Issues: []app.Item{}}, nil
	} else if err != nil {
		return app.Cart{}, fmt.Errorf("read cart: %w", err)
	}
	updated := make([]storedCartItem, 0, len(stored.Items))
	for _, item := range stored.Items {
		if item.ProductID != productOID || item.SKU != sku {
			updated = append(updated, item)
		}
	}
	if err := r.saveItems(ctx, stored, updated, ""); err != nil {
		return app.Cart{}, err
	}
	return r.hydrate(ctx, updated)
}

func (r *CartRepository) Merge(ctx context.Context, userID, requestID string, selections []app.Selection) (app.Cart, error) {
	customerID, err := cartObjectID(userID)
	if err != nil {
		return app.Cart{}, err
	}
	stored, err := r.loadOrCreate(ctx, customerID)
	if err != nil {
		return app.Cart{}, err
	}
	if stored.LastMergeRequestID == requestID {
		return r.hydrate(ctx, stored.Items)
	}
	updated := append([]storedCartItem(nil), stored.Items...)
	issues := make([]app.Item, 0)
	for _, selection := range selections {
		product, issue, validationErr := r.selectionProduct(ctx, selection)
		if validationErr != nil {
			return app.Cart{}, validationErr
		}
		if issue != "" {
			issues = append(issues, cartIssue(selection, issue))
			continue
		}
		updated = mergeStoredItem(updated, selection, product.Price)
	}
	if err := r.saveItems(ctx, stored, updated, requestID); err != nil {
		return app.Cart{}, err
	}
	result, err := r.hydrate(ctx, updated)
	if err != nil {
		return app.Cart{}, err
	}
	result.Issues = append(result.Issues, issues...)
	return result, nil
}

func (r *CartRepository) Validate(ctx context.Context, selections []app.Selection) (app.Cart, error) {
	items := make([]storedCartItem, 0, len(selections))
	for _, selection := range selections {
		if _, _, err := r.selectionProduct(ctx, selection); err != nil {
			return app.Cart{}, err
		}
		items = append(items, storedFromSelection(selection, 0))
	}
	return r.hydrate(ctx, items)
}

func (r *CartRepository) loadOrCreate(ctx context.Context, customerID bson.ObjectID) (storedCart, error) {
	var stored storedCart
	err := r.carts.FindOne(ctx, bson.M{"customer_id": customerID}).Decode(&stored)
	if err == nil {
		return stored, nil
	}
	if !errors.Is(err, driver.ErrNoDocuments) {
		return storedCart{}, fmt.Errorf("read cart: %w", err)
	}
	now := time.Now().UTC()
	stored = storedCart{ID: bson.NewObjectID(), CustomerID: customerID, Items: []storedCartItem{}, Version: 1}
	_, err = r.carts.InsertOne(ctx, bson.M{"_id": stored.ID, "customer_id": customerID, "items": stored.Items, "version": stored.Version, "created_at": now, "updated_at": now})
	if driver.IsDuplicateKeyError(err) {
		return r.loadOrCreate(ctx, customerID)
	}
	return stored, err
}

func (r *CartRepository) saveItems(ctx context.Context, stored storedCart, items []storedCartItem, mergeID string) error {
	set := bson.M{"items": items, "updated_at": time.Now().UTC()}
	if mergeID != "" {
		set["last_merge_request_id"] = mergeID
	}
	result, err := r.carts.UpdateOne(ctx, cartVersionFilter(stored.ID, stored.Version), bson.M{"$set": set, "$inc": bson.M{"version": 1}})
	if err != nil {
		return fmt.Errorf("save cart: %w", err)
	}
	if result.MatchedCount == 0 {
		return app.ErrConflict
	}
	return nil
}

func mergeStoredItem(items []storedCartItem, selection app.Selection, unitPrice int64) []storedCartItem {
	productID, _ := bson.ObjectIDFromHex(selection.ProductID)
	for i := range items {
		if items[i].ProductID == productID && items[i].SKU == selection.SKU {
			items[i].Quantity += selection.Quantity
			items[i].UnitPrice = unitPrice
			return items
		}
	}
	return append(items, storedFromSelection(selection, unitPrice))
}

func replaceStoredItem(items []storedCartItem, selection app.Selection, unitPrice int64) []storedCartItem {
	productID, _ := bson.ObjectIDFromHex(selection.ProductID)
	for i := range items {
		if items[i].ProductID == productID && items[i].SKU == selection.SKU {
			items[i].Quantity = selection.Quantity
			items[i].UnitPrice = unitPrice
			return items
		}
	}
	return items
}

func containsStoredItem(items []storedCartItem, selection app.Selection) bool {
	productID, _ := bson.ObjectIDFromHex(selection.ProductID)
	for _, item := range items {
		if item.ProductID == productID && item.SKU == selection.SKU {
			return true
		}
	}
	return false
}

func storedFromSelection(selection app.Selection, unitPrice int64) storedCartItem {
	productID, _ := bson.ObjectIDFromHex(selection.ProductID)
	return storedCartItem{ProductID: productID, SKU: selection.SKU, Quantity: selection.Quantity, ColorName: selection.Color, FulfillmentMode: "ready", UnitPrice: unitPrice, AddedAt: time.Now().UTC()}
}

func (r *CartRepository) selectionProduct(ctx context.Context, selection app.Selection) (visibleProduct, string, error) {
	products, err := r.visibleProducts(ctx, []string{selection.ProductID})
	if err != nil {
		return visibleProduct{}, "", err
	}
	product, ok := products[selection.ProductID]
	if !ok {
		return visibleProduct{}, app.IssueProductUnavailable, nil
	}
	variant, ok := product.Variants[selection.SKU]
	if len(product.Variants) == 0 && product.SKU == selection.SKU {
		if product.Stock <= 0 {
			return product, app.IssueOutOfStock, nil
		}
		if selection.Quantity > product.Stock {
			return product, app.IssueQuantityExceeded, nil
		}
		return product, "", nil
	}
	if !ok {
		return product, app.IssueSKUNotFound, nil
	}
	if (selection.Color != "" && !strings.EqualFold(selection.Color, variant.Color)) || (selection.Size != "" && !strings.EqualFold(selection.Size, variant.Size)) {
		return product, app.IssueSKUNotFound, nil
	}
	if product.Stock <= 0 || variant.Stock <= 0 {
		return product, app.IssueOutOfStock, nil
	}
	if selection.Quantity > variant.Stock || selection.Quantity > product.Stock {
		return product, app.IssueQuantityExceeded, nil
	}
	return product, "", nil
}

func (r *CartRepository) hydrate(ctx context.Context, stored []storedCartItem) (app.Cart, error) {
	ids := make([]string, 0, len(stored))
	for _, item := range stored {
		ids = append(ids, item.ProductID.Hex())
	}
	products, err := r.visibleProducts(ctx, ids)
	if err != nil {
		return app.Cart{}, err
	}
	result := app.Cart{Items: []app.Item{}, Issues: []app.Item{}}
	for _, storedItem := range stored {
		selection := app.Selection{ProductID: storedItem.ProductID.Hex(), SKU: storedItem.SKU, Color: storedItem.ColorName, Quantity: storedItem.Quantity, FulfillmentMode: "ready"}
		item := app.Item{Selection: selection, UnitPrice: storedItem.UnitPrice}
		product, ok := products[storedItem.ProductID.Hex()]
		if !ok {
			item.Issue, item.IssueDetail = app.IssueProductUnavailable, "Sản phẩm không còn được bán"
		} else if len(product.Variants) == 0 && product.SKU == storedItem.SKU {
			item.ProductSlug, item.Name, item.Image, item.UnitPrice, item.Stock = product.Slug, product.Name, product.Image, product.Price, product.Stock
			item.Available = product.Stock > 0 && storedItem.Quantity <= product.Stock
			if !item.Available {
				item.Issue, item.IssueDetail = app.IssueQuantityExceeded, "Số lượng vượt tồn kho hiện tại"
			} else if storedItem.UnitPrice > 0 && storedItem.UnitPrice != product.Price {
				item.Issue, item.IssueDetail = app.IssuePriceChanged, "Giá sản phẩm đã thay đổi"
			}
			item.LineTotal = item.UnitPrice * storedItem.Quantity
		} else if variant, exists := product.Variants[storedItem.SKU]; !exists {
			item.Issue, item.IssueDetail = app.IssueSKUNotFound, "SKU không còn tồn tại"
		} else {
			item.ProductSlug, item.Name, item.Image, item.UnitPrice, item.Stock = product.Slug, product.Name, product.Image, product.Price, variant.Stock
			item.Color, item.Size = variant.Color, variant.Size
			item.Available = product.Stock > 0 && variant.Stock > 0 && storedItem.Quantity <= variant.Stock && storedItem.Quantity <= product.Stock
			if !item.Available {
				item.Issue, item.IssueDetail = app.IssueQuantityExceeded, "Số lượng vượt tồn kho hiện tại"
			} else if storedItem.UnitPrice > 0 && storedItem.UnitPrice != product.Price {
				item.Issue, item.IssueDetail = app.IssuePriceChanged, "Giá sản phẩm đã thay đổi"
			}
			item.LineTotal = item.UnitPrice * storedItem.Quantity
		}
		if item.Issue != "" {
			result.Issues = append(result.Issues, item)
		} else {
			result.Items = append(result.Items, item)
			result.ItemCount += storedItem.Quantity
			result.Subtotal += item.LineTotal
		}
	}
	return result, nil
}

func cartVersionFilter(id bson.ObjectID, version int64) bson.M {
	return bson.M{"_id": id, "$or": bson.A{bson.M{"version": version}, bson.M{"version": bson.M{"$exists": false}}}}
}

func cartIssue(selection app.Selection, issue string) app.Item {
	detail := "Sản phẩm hoặc SKU không còn phù hợp"
	switch issue {
	case app.IssueProductUnavailable:
		detail = "Sản phẩm không còn được bán"
	case app.IssueSKUNotFound:
		detail = "SKU không còn tồn tại"
	case app.IssueOutOfStock:
		detail = "Sản phẩm đã hết hàng"
	case app.IssueQuantityExceeded:
		detail = "Số lượng vượt tồn kho hiện tại"
	}
	return app.Item{Selection: selection, Issue: issue, IssueDetail: detail}
}

func (r *CartRepository) visibleProducts(ctx context.Context, ids []string) (map[string]visibleProduct, error) {
	objectIDs := make([]bson.ObjectID, 0, len(ids))
	for _, id := range ids {
		if oid, err := bson.ObjectIDFromHex(id); err == nil {
			objectIDs = append(objectIDs, oid)
		}
	}
	result := make(map[string]visibleProduct)
	if len(objectIDs) == 0 {
		return result, nil
	}
	pipeline := bson.A{
		bson.M{"$match": bson.M{"_id": bson.M{"$in": objectIDs}, "is_active": true}},
		bson.M{"$lookup": bson.M{"from": "categories", "localField": "category_id", "foreignField": "_id", "as": "_category"}},
		bson.M{"$unwind": "$_category"},
		bson.M{"$graphLookup": bson.M{"from": "categories", "startWith": "$_category.parent_id", "connectFromField": "parent_id", "connectToField": "_id", "as": "_ancestors"}},
		bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
			bson.M{"$eq": bson.A{"$_category.is_active", true}},
			bson.M{"$allElementsTrue": bson.A{bson.M{"$map": bson.M{"input": "$_ancestors", "as": "ancestor", "in": bson.M{"$eq": bson.A{"$$ancestor.is_active", true}}}}}},
			bson.M{"$or": bson.A{
				bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$_category.parent_id", nil}}, nil}},
				bson.M{"$gt": bson.A{
					bson.M{"$size": bson.M{"$filter": bson.M{
						"input": "$_ancestors", "as": "ancestor",
						"cond": bson.M{"$eq": bson.A{bson.M{"$ifNull": bson.A{"$$ancestor.parent_id", nil}}, nil}},
					}}},
					0,
				}},
			}},
		}}}},
		bson.M{"$project": bson.M{"_id": 1, "slug": 1, "name": 1, "sku": 1, "image": 1, "price": 1, "stock": 1, "variants": 1}},
	}
	cursor, err := r.products.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var raw bson.M
		if err := cursor.Decode(&raw); err != nil {
			return nil, err
		}
		product := visibleProduct{ID: asID(raw["_id"]), Slug: asString(raw["slug"]), Name: asString(raw["name"]), SKU: asString(raw["sku"]), Image: asString(asMap(raw["image"])["src"]), Price: asInt64(raw["price"]), Stock: asInt64(raw["stock"]), Variants: map[string]visibleVariant{}}
		for _, rawVariant := range asSlice(raw["variants"]) {
			variant := asMap(rawVariant)
			sku := asString(variant["sku"])
			product.Variants[sku] = visibleVariant{Color: asString(variant["color"]), Size: asString(variant["size"]), Stock: asInt64(variant["stock"])}
		}
		result[product.ID] = product
	}
	return result, cursor.Err()
}

func cartObjectID(value string) (bson.ObjectID, error) {
	id, err := bson.ObjectIDFromHex(value)
	if err != nil {
		return bson.ObjectID{}, app.Invalid{Field: "customer_id", Message: "ID khách hàng không hợp lệ"}
	}
	return id, nil
}
