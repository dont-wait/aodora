package mongo

import (
	"context"
	"errors"
	"strings"
	"time"

	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// TailoringOrderRepository creates custom orders from tailoring requests.
//
// Duplicate protection (no migration, no unique index on tailoring_request_id):
// inside ONE snapshot transaction the request document is re-read, an existing
// order for it is looked up, and — if none — the order is inserted and the
// request's updated_at is advanced with a filter on the values just read. Two
// concurrent transactions therefore both write the same request document, so
// MongoDB aborts one with a WriteConflict; WithTransaction re-runs it, it then
// sees the committed order and returns it with created=false. Writers that bypass
// this code path are not blocked by the database (see docs: optional unique
// partial index needs a new migration).
type TailoringOrderRepository struct {
	client   *driver.Client
	requests *driver.Collection
	orders   *driver.Collection
}

func NewTailoringOrderRepository(client *driver.Client, db *driver.Database) *TailoringOrderRepository {
	return &TailoringOrderRepository{client: client, requests: db.Collection("tailoring_requests"), orders: db.Collection("orders")}
}

func (r *TailoringOrderRepository) findByTailoringID(ctx context.Context, oid bson.ObjectID) (*domainorder.Order, error) {
	var doc bson.M
	if err := r.orders.FindOne(ctx, bson.M{"tailoring_request_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	o, err := decodeOrder(doc)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *TailoringOrderRepository) FindCustomOrder(ctx context.Context, tailoringRequestID string) (*domainorder.Order, error) {
	oid, err := bson.ObjectIDFromHex(tailoringRequestID)
	if err != nil {
		return nil, nil
	}
	return r.findByTailoringID(ctx, oid)
}

func (r *TailoringOrderRepository) CreateCustomOrder(ctx context.Context, d apptailoring.CustomOrderDraft) (*domainorder.Order, bool, error) {
	reqOID, err := bson.ObjectIDFromHex(d.TailoringRequestID)
	if err != nil {
		return nil, false, apptailoring.ErrNotFound
	}
	prodOID, err := bson.ObjectIDFromHex(d.ProductID)
	if err != nil {
		return nil, false, err
	}
	sess, err := r.client.StartSession()
	if err != nil {
		return nil, false, err
	}
	defer sess.EndSession(ctx)

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var (
		result  *domainorder.Order
		created bool
	)
	txOpts := options.Transaction().SetReadConcern(readconcern.Snapshot()).SetWriteConcern(writeconcern.Majority())
	_, err = sess.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		result, created = nil, false // the callback may be re-run after a WriteConflict

		var req storedTailoringRequest
		if err := r.requests.FindOne(txCtx, bson.M{"_id": reqOID}).Decode(&req); err != nil {
			if errors.Is(err, driver.ErrNoDocuments) {
				return nil, apptailoring.ErrNotFound
			}
			return nil, err
		}
		existing, err := r.findByTailoringID(txCtx, reqOID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			result = existing
			return nil, nil
		}
		if req.Status != domaintailoring.StatusMeasurementConfirmed && req.Status != domaintailoring.StatusInProgress {
			return nil, apptailoring.ErrStateConflict
		}
		dom := req.toDomain()
		if !domaintailoring.HasAllCMMeasurements(dom.Measurements) {
			return nil, apptailoring.ErrMeasurementIncomplete
		}

		subtotal, err := encodeMoney(d.Subtotal)
		if err != nil {
			return nil, err
		}
		fee, err := encodeMoney(d.ShippingFee)
		if err != nil {
			return nil, err
		}
		total, err := encodeMoney(d.Total)
		if err != nil {
			return nil, err
		}
		unit, err := encodeMoney(d.UnitPrice)
		if err != nil {
			return nil, err
		}
		addr := bson.M{"recipient": d.Address.Recipient, "phone": d.Address.Phone}
		for k, v := range map[string]string{"line1": d.Address.Line1, "ward": d.Address.Ward, "district": d.Address.District, "city": d.Address.City} {
			if v != "" {
				addr[k] = v
			}
		}
		// Snapshot copied by value: later edits of the request never touch the order.
		profile := *req.Measurements
		orderID := bson.NewObjectID()
		doc := bson.M{
			"_id":                  orderID,
			"order_number":         d.OrderNumber,
			"status":               domainorder.StatusPending,
			"payment_method":       d.PaymentMethod,
			"payment_status":       domainorder.PaymentUnpaid,
			"subtotal":             subtotal,
			"shipping_fee":         fee,
			"total":                total,
			"fulfillment_mode":     "custom",
			"tailoring_request_id": reqOID,
			"measurement_profile":  profile,
			"shipping_address":     addr,
			"items": bson.A{bson.M{
				"product_id":       prodOID,
				"product_name":     d.ProductName,
				"sku":              apptailoring.CustomSKU,
				"quantity":         d.Quantity,
				"unit_price":       unit,
				"fulfillment_mode": "custom",
				"measurements":     profile,
			}},
			"created_at": d.Now,
			"updated_at": d.Now,
		}
		if req.CustomerID != nil {
			doc["customer_id"] = *req.CustomerID // guest request: field omitted, no fake customer
		}
		if d.Note != "" {
			doc["note"] = d.Note
		}
		// No checkout marker: the validator only requires it together with
		// scope/guest-hash rules; custom orders are admin-created.
		if _, err := r.orders.InsertOne(txCtx, doc); err != nil {
			if driver.IsDuplicateKeyError(err) && strings.Contains(err.Error(), "uq_orders_order_number") {
				return nil, apptailoring.ErrDuplicateOrderNumber
			}
			return nil, err
		}
		bumped := d.Now.Truncate(time.Millisecond)
		if !bumped.After(req.UpdatedAt) {
			bumped = req.UpdatedAt.Add(time.Millisecond)
		}
		res, err := r.requests.UpdateOne(txCtx,
			bson.M{"_id": reqOID, "status": req.Status, "updated_at": req.UpdatedAt},
			bson.M{"$set": bson.M{"updated_at": bumped}})
		if err != nil {
			return nil, err
		}
		if res.MatchedCount == 0 {
			return nil, apptailoring.ErrConcurrentUpdate
		}
		var stored bson.M
		if err := r.orders.FindOne(txCtx, bson.M{"_id": orderID}).Decode(&stored); err != nil {
			return nil, err
		}
		o, err := decodeOrder(stored)
		if err != nil {
			return nil, err
		}
		result, created = &o, true
		return nil, nil
	}, txOpts)
	if err != nil {
		return nil, false, err
	}
	return result, created, nil
}
