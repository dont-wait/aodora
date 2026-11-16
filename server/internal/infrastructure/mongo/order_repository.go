package mongo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

type OrderRepository struct {
	client   *driver.Client
	db       *driver.Database
	products *driver.Collection
	orders   *driver.Collection
}

func NewOrderRepository(client *driver.Client, db *driver.Database) *OrderRepository {
	return &OrderRepository{
		client:   client,
		db:       db,
		products: db.Collection("products"),
		orders:   db.Collection("orders"),
	}
}

func decodeMoney(val any) (domainorder.Money, error) {
	switch v := val.(type) {
	case int32:
		if v < 0 {
			return domainorder.Money{}, errors.New("negative money not allowed")
		}
		return domainorder.MoneyFromCoefficient(big.NewInt(int64(v)), 0)
	case int64:
		if v < 0 {
			return domainorder.Money{}, errors.New("negative money not allowed")
		}
		return domainorder.MoneyFromCoefficient(big.NewInt(v), 0)
	case int:
		if v < 0 {
			return domainorder.Money{}, errors.New("negative money not allowed")
		}
		return domainorder.MoneyFromCoefficient(big.NewInt(int64(v)), 0)
	case bson.Decimal128:
		if v.IsNaN() || v.IsInf() != 0 {
			return domainorder.Money{}, errors.New("NaN or Inf decimal128 not allowed")
		}
		bi, exp, err := v.BigInt()
		if err != nil {
			return domainorder.Money{}, err
		}
		if bi.Sign() < 0 {
			return domainorder.Money{}, errors.New("negative money not allowed")
		}
		return domainorder.MoneyFromCoefficient(bi, exp)
	default:
		return domainorder.Money{}, fmt.Errorf("unsupported money BSON type %T", val)
	}
}

func encodeMoney(m domainorder.Money) (any, error) {
	if i64, ok := m.Int64(); ok {
		return i64, nil
	}
	return bson.ParseDecimal128(m.String())
}

// SellableProducts returns public sellable products with active categories and ancestors.
func (r *OrderRepository) SellableProducts(ctx context.Context, productIDs []string) (map[string]apporder.Product, error) {
	if len(productIDs) == 0 {
		return map[string]apporder.Product{}, nil
	}
	oids := make([]bson.ObjectID, 0, len(productIDs))
	for _, idStr := range productIDs {
		id, err := oid(idStr)
		if err != nil {
			continue
		}
		oids = append(oids, id)
	}
	if len(oids) == 0 {
		return map[string]apporder.Product{}, nil
	}

	pipeline := bson.A{
		bson.M{"$match": bson.M{
			"_id":       bson.M{"$in": oids},
			"is_active": true,
		}},
	}
	pipeline = append(pipeline, categoryVisibilityStages()...)

	cursor, err := r.products.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	result := make(map[string]apporder.Product)
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		idHex := asID(doc["_id"])
		name := asString(doc["name"])
		sku := asString(doc["sku"])
		price, err := decodeMoney(doc["price"])
		if err != nil {
			return nil, fmt.Errorf("decode product price for %s: %w", idHex, err)
		}
		stock := asInt64(doc["stock"])

		var variants []apporder.Variant
		for _, v := range asSlice(doc["variants"]) {
			vm := asMap(v)
			variants = append(variants, apporder.Variant{
				SKU:   asString(vm["sku"]),
				Color: asString(vm["color"]),
				Size:  asString(vm["size"]),
				Stock: asInt64(vm["stock"]),
			})
		}

		result[idHex] = apporder.Product{
			ID:       idHex,
			Name:     name,
			SKU:      sku,
			Price:    price,
			Stock:    stock,
			Variants: variants,
		}
	}
	return result, cursor.Err()
}

type orderCheckoutTx struct {
	repo *OrderRepository
}

func (tx *orderCheckoutTx) SellableProducts(ctx context.Context, productIDs []string) (map[string]apporder.Product, error) {
	return tx.repo.SellableProducts(ctx, productIDs)
}

func (tx *orderCheckoutTx) DecrementStock(ctx context.Context, lines []apporder.StockLine) error {
	type productGroup struct {
		productID     string
		hasVariants   bool
		totalQty      int64
		skuQuantities []struct {
			sku string
			qty int64
		}
	}

	groups := make(map[string]*productGroup)
	var orderOfProds []string

	for _, line := range lines {
		g, exists := groups[line.ProductID]
		if !exists {
			g = &productGroup{
				productID:   line.ProductID,
				hasVariants: line.HasVariants,
			}
			groups[line.ProductID] = g
			orderOfProds = append(orderOfProds, line.ProductID)
		}
		g.totalQty += line.Quantity
		g.skuQuantities = append(g.skuQuantities, struct {
			sku string
			qty int64
		}{sku: line.SKU, qty: line.Quantity})
	}

	for _, prodID := range orderOfProds {
		g := groups[prodID]
		pID, err := oid(g.productID)
		if err != nil {
			return apporder.ErrOutOfStock
		}

		if g.hasVariants {
			elemMatches := make(bson.A, len(g.skuQuantities))
			incFields := bson.M{"stock": -g.totalQty}
			var arrayFilters []any

			for i, sq := range g.skuQuantities {
				elemMatches[i] = bson.M{
					"$elemMatch": bson.M{
						"sku":   sq.sku,
						"stock": bson.M{"$gte": sq.qty},
					},
				}
				tag := fmt.Sprintf("v%d", i)
				incFields[fmt.Sprintf("variants.$[%s].stock", tag)] = -sq.qty
				arrayFilters = append(arrayFilters, bson.M{fmt.Sprintf("%s.sku", tag): sq.sku})
			}

			filter := bson.M{
				"_id":       pID,
				"is_active": true,
				"stock":     bson.M{"$gte": g.totalQty},
				"variants":  bson.M{"$all": elemMatches},
				"$expr":     bson.M{"$eq": bson.A{bson.M{"$sum": "$variants.stock"}, "$stock"}},
			}

			update := bson.M{
				"$inc": incFields,
				"$set": bson.M{"updated_at": time.Now().UTC()},
			}

			opts := options.UpdateOne().SetArrayFilters(arrayFilters)
			res, err := tx.repo.products.UpdateOne(ctx, filter, update, opts)
			if err != nil {
				return err
			}
			if res.MatchedCount == 0 {
				return tx.diagnoseFailure(ctx, pID)
			}
		} else {
			// Legacy product without variants
			filter := bson.M{
				"_id":       pID,
				"is_active": true,
				"stock":     bson.M{"$gte": g.totalQty},
				"$or": bson.A{
					bson.M{"variants": bson.M{"$exists": false}},
					bson.M{"variants": bson.M{"$size": 0}},
				},
			}
			if len(g.skuQuantities) > 0 {
				filter["sku"] = g.skuQuantities[0].sku
			}
			update := bson.M{
				"$inc": bson.M{"stock": -g.totalQty},
				"$set": bson.M{"updated_at": time.Now().UTC()},
			}
			res, err := tx.repo.products.UpdateOne(ctx, filter, update)
			if err != nil {
				return err
			}
			if res.MatchedCount == 0 {
				return tx.diagnoseFailure(ctx, pID)
			}
		}
	}
	return nil
}

func (tx *orderCheckoutTx) diagnoseFailure(ctx context.Context, pID bson.ObjectID) error {
	var doc bson.M
	if err := tx.repo.products.FindOne(ctx, bson.M{"_id": pID}).Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return apporder.ErrOutOfStock
		}
		return err
	}

	variants := asSlice(doc["variants"])
	if len(variants) > 0 {
		var sumStock int64
		for _, v := range variants {
			sumStock += asInt64(asMap(v)["stock"])
		}
		if sumStock != asInt64(doc["stock"]) {
			return apporder.ErrStockInconsistent
		}
	}
	return apporder.ErrOutOfStock
}

func (tx *orderCheckoutTx) InsertOrder(ctx context.Context, o domainorder.Order) (domainorder.Order, error) {
	doc := bson.M{
		"_id":            bson.NewObjectID(),
		"order_number":   o.OrderNumber,
		"status":         o.Status,
		"payment_method": o.PaymentMethod,
		"payment_status": o.PaymentStatus,
	}

	if o.CustomerID != "" {
		custOID, err := oid(o.CustomerID)
		if err != nil {
			return domainorder.Order{}, err
		}
		doc["customer_id"] = custOID
	} else {
		doc["guest_access_hash"] = o.GuestAccessHash
	}

	subtotal, err := encodeMoney(o.Subtotal)
	if err != nil {
		return domainorder.Order{}, err
	}
	shippingFee, err := encodeMoney(o.ShippingFee)
	if err != nil {
		return domainorder.Order{}, err
	}
	total, err := encodeMoney(o.Total)
	if err != nil {
		return domainorder.Order{}, err
	}

	doc["subtotal"] = subtotal
	doc["shipping_fee"] = shippingFee
	doc["total"] = total
	doc["fulfillment_mode"] = "ready"

	addr := bson.M{
		"recipient": o.ShippingAddress.Recipient,
		"phone":     o.ShippingAddress.Phone,
	}
	if o.ShippingAddress.Line1 != "" {
		addr["line1"] = o.ShippingAddress.Line1
	}
	if o.ShippingAddress.Ward != "" {
		addr["ward"] = o.ShippingAddress.Ward
	}
	if o.ShippingAddress.District != "" {
		addr["district"] = o.ShippingAddress.District
	}
	if o.ShippingAddress.City != "" {
		addr["city"] = o.ShippingAddress.City
	}
	doc["shipping_address"] = addr

	items := make(bson.A, len(o.Items))
	for i, it := range o.Items {
		pID, err := oid(it.ProductID)
		if err != nil {
			return domainorder.Order{}, err
		}
		uPrice, err := encodeMoney(it.UnitPrice)
		if err != nil {
			return domainorder.Order{}, err
		}
		itDoc := bson.M{
			"product_id":       pID,
			"product_name":     it.ProductName,
			"sku":              it.SKU,
			"quantity":         it.Quantity,
			"unit_price":       uPrice,
			"fulfillment_mode": "ready",
		}
		if it.Color != "" {
			itDoc["color"] = it.Color
		}
		if it.Size != "" {
			itDoc["size"] = it.Size
		}
		items[i] = itDoc
	}
	doc["items"] = items

	if o.Note != "" {
		doc["note"] = o.Note
	}

	doc["checkout"] = bson.M{
		"scope":        o.Checkout.Scope,
		"request_id":   o.Checkout.RequestID,
		"request_hash": o.Checkout.RequestHash,
	}

	doc["created_at"] = o.CreatedAt
	doc["updated_at"] = o.UpdatedAt

	_, err = tx.repo.orders.InsertOne(ctx, doc)
	if err != nil {
		if driver.IsDuplicateKeyError(err) {
			errStr := err.Error()
			if strings.Contains(errStr, "uq_orders_checkout_scope_request") {
				return domainorder.Order{}, apporder.ErrDuplicateAttempt
			}
			if strings.Contains(errStr, "uq_orders_order_number") {
				return domainorder.Order{}, apporder.ErrDuplicateOrderNumber
			}
		}
		return domainorder.Order{}, err
	}

	o.ID = doc["_id"].(bson.ObjectID).Hex()
	return o, nil
}

func (r *OrderRepository) RunCheckout(ctx context.Context, fn func(ctx context.Context, tx apporder.CheckoutTx) error) error {
	sess, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("%w: %w", apporder.ErrOrderUnavailable, err)
	}
	defer sess.EndSession(ctx)

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	txOpts := options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(writeconcern.Majority())

	_, err = sess.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		tx := &orderCheckoutTx{
			repo: r,
		}
		err := fn(sessCtx, tx)
		return nil, err
	}, txOpts)

	if err == nil {
		return nil
	}

	// Standalone server rejection
	var cmdErr driver.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Code == 20 {
		return fmt.Errorf("%w: %w", apporder.ErrOrderUnavailable, err)
	}
	if strings.Contains(err.Error(), "Transaction numbers are only allowed on a replica set member or mongos") {
		return fmt.Errorf("%w: %w", apporder.ErrOrderUnavailable, err)
	}

	return err
}

func decodeOrder(doc bson.M) (domainorder.Order, error) {
	idHex := asID(doc["_id"])
	orderNum := asString(doc["order_number"])
	custID := ""
	if doc["customer_id"] != nil {
		custID = asID(doc["customer_id"])
	}
	guestHash := asString(doc["guest_access_hash"])
	status := asString(doc["status"])
	payMethod := asString(doc["payment_method"])
	payStatus := asString(doc["payment_status"])

	subtotal, err := decodeMoney(doc["subtotal"])
	if err != nil {
		return domainorder.Order{}, fmt.Errorf("decode subtotal: %w", err)
	}
	shippingFee, err := decodeMoney(doc["shipping_fee"])
	if err != nil {
		return domainorder.Order{}, fmt.Errorf("decode shipping_fee: %w", err)
	}
	total, err := decodeMoney(doc["total"])
	if err != nil {
		return domainorder.Order{}, fmt.Errorf("decode total: %w", err)
	}

	addrDoc := asMap(doc["shipping_address"])
	addr := domainorder.NormalizedAddress{
		Recipient: asString(addrDoc["recipient"]),
		Phone:     asString(addrDoc["phone"]),
		Line1:     asString(addrDoc["line1"]),
		Ward:      asString(addrDoc["ward"]),
		District:  asString(addrDoc["district"]),
		City:      asString(addrDoc["city"]),
	}

	var items []domainorder.OrderItem
	for _, itVal := range asSlice(doc["items"]) {
		itDoc := asMap(itVal)
		uPrice, err := decodeMoney(itDoc["unit_price"])
		if err != nil {
			return domainorder.Order{}, fmt.Errorf("decode item unit_price: %w", err)
		}
		items = append(items, domainorder.OrderItem{
			ProductID:   asID(itDoc["product_id"]),
			ProductName: asString(itDoc["product_name"]),
			SKU:         asString(itDoc["sku"]),
			Color:       asString(itDoc["color"]),
			Size:        asString(itDoc["size"]),
			Quantity:    asInt64(itDoc["quantity"]),
			UnitPrice:   uPrice,
		})
	}

	note := asString(doc["note"])

	var checkout domainorder.OrderCheckout
	if chkVal := doc["checkout"]; chkVal != nil {
		chkDoc := asMap(chkVal)
		checkout = domainorder.OrderCheckout{
			Scope:       asString(chkDoc["scope"]),
			RequestID:   asString(chkDoc["request_id"]),
			RequestHash: asString(chkDoc["request_hash"]),
		}
	}

	createdAt := asTime(doc["created_at"])
	updatedAt := asTime(doc["updated_at"])

	return domainorder.Order{
		ID:              idHex,
		OrderNumber:     orderNum,
		CustomerID:      custID,
		GuestAccessHash: guestHash,
		Status:          status,
		PaymentMethod:   payMethod,
		PaymentStatus:   payStatus,
		Subtotal:        subtotal,
		ShippingFee:     shippingFee,
		Total:           total,
		ShippingAddress: addr,
		Items:           items,
		Note:            note,
		Checkout:        checkout,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

func (r *OrderRepository) FindByAttempt(ctx context.Context, scope, requestID string) (*domainorder.Order, error) {
	var doc bson.M
	filter := bson.M{
		"checkout.scope":      scope,
		"checkout.request_id": requestID,
	}
	err := r.orders.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	order, err := decodeOrder(doc)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id string) (*domainorder.Order, error) {
	itemID, err := oid(id)
	if err != nil {
		return nil, nil
	}
	var doc bson.M
	err = r.orders.FindOne(ctx, bson.M{"_id": itemID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	order, err := decodeOrder(doc)
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) ListByCustomer(ctx context.Context, customerID string, page, pageSize int64) ([]domainorder.Order, int64, error) {
	custOID, err := oid(customerID)
	if err != nil {
		return nil, 0, nil
	}
	filter := bson.M{"customer_id": custOID}
	total, err := r.orders.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	skip := (page - 1) * pageSize
	findOpts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(skip).
		SetLimit(pageSize)

	cursor, err := r.orders.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var orders []domainorder.Order
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, 0, err
		}
		order, err := decodeOrder(doc)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, order)
	}
	return orders, total, cursor.Err()
}
