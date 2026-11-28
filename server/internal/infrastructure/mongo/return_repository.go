package mongo

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	returnsapp "github.com/your-org/aodora/server/internal/application/returns"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	dr "github.com/your-org/aodora/server/internal/domain/returns"
)

// ReturnRepository stores return_requests; restock and refund reuse the order repository.
type ReturnRepository struct {
	returns *driver.Collection
	orders  *OrderRepository
}

func NewReturnRepository(orders *OrderRepository) *ReturnRepository {
	return &ReturnRepository{returns: orders.db.Collection("return_requests"), orders: orders}
}

type returnDoc struct {
	ID           bson.ObjectID `bson:"_id"`
	OrderID      bson.ObjectID `bson:"order_id"`
	OrderNumber  string        `bson:"order_number"`
	CustomerID   bson.ObjectID `bson:"customer_id"`
	CustomerName string        `bson:"customer_name"`
	Phone        string        `bson:"phone"`
	Items        []dr.Item     `bson:"items"`
	Reason       string        `bson:"reason"`
	Note         string        `bson:"note"`
	Evidence     []dr.Evidence `bson:"evidence"`
	Status       string        `bson:"status"`
	// open is the order id while the request is open, else absent; a unique
	// partial index on it allows one open request per order.
	Open         *bson.ObjectID `bson:"open_order_id,omitempty"`
	RefundAmount int64          `bson:"refund_amount"`
	Restocked    bool           `bson:"restocked"`
	AdminNote    string         `bson:"admin_note"`
	History      []dr.Event     `bson:"history"`
	DeliveredAt  time.Time      `bson:"delivered_at"`
	CreatedAt    time.Time      `bson:"created_at"`
	UpdatedAt    time.Time      `bson:"updated_at"`
}

func toReturnDoc(r dr.Request) (returnDoc, error) {
	orderID, err := oid(r.OrderID)
	if err != nil {
		return returnDoc{}, returnsapp.ErrNotFound
	}
	customerID, err := oid(r.CustomerID)
	if err != nil {
		return returnDoc{}, returnsapp.ErrNotFound
	}
	id := bson.NewObjectID()
	if r.ID != "" {
		if id, err = oid(r.ID); err != nil {
			return returnDoc{}, returnsapp.ErrNotFound
		}
	}
	d := returnDoc{ID: id, OrderID: orderID, OrderNumber: r.OrderNumber, CustomerID: customerID, CustomerName: r.CustomerName, Phone: r.Phone,
		Items: r.Items, Reason: r.Reason, Note: r.Note, Evidence: r.Evidence, Status: r.Status, RefundAmount: r.RefundAmount,
		Restocked: r.Restocked, AdminNote: r.AdminNote, History: r.History, DeliveredAt: r.DeliveredAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if dr.Open(r.Status) {
		d.Open = &orderID
	}
	return d, nil
}

func (d returnDoc) request() dr.Request {
	return dr.Request{ID: d.ID.Hex(), OrderID: d.OrderID.Hex(), OrderNumber: d.OrderNumber, CustomerID: d.CustomerID.Hex(), CustomerName: d.CustomerName, Phone: d.Phone,
		Items: nonNil(d.Items), Reason: d.Reason, Note: d.Note, Evidence: nonNil(d.Evidence), Status: d.Status, RefundAmount: d.RefundAmount,
		Restocked: d.Restocked, AdminNote: d.AdminNote, History: nonNil(d.History), DeliveredAt: d.DeliveredAt.UTC(), CreatedAt: d.CreatedAt.UTC(), UpdatedAt: d.UpdatedAt.UTC()}
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (r *ReturnRepository) Create(ctx context.Context, req dr.Request) (dr.Request, error) {
	d, err := toReturnDoc(req)
	if err != nil {
		return dr.Request{}, err
	}
	if _, err := r.returns.InsertOne(ctx, d); err != nil {
		if driver.IsDuplicateKeyError(err) {
			return dr.Request{}, returnsapp.ErrConflict
		}
		return dr.Request{}, err
	}
	return d.request(), nil
}

func (r *ReturnRepository) Get(ctx context.Context, id string) (*dr.Request, error) {
	objectID, err := oid(id)
	if err != nil {
		return nil, nil
	}
	var d returnDoc
	err = r.returns.FindOne(ctx, bson.M{"_id": objectID}).Decode(&d)
	if errors.Is(err, driver.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := d.request()
	return &out, nil
}

func (r *ReturnRepository) find(ctx context.Context, filter bson.M, opts *options.FindOptionsBuilder) ([]dr.Request, error) {
	cur, err := r.returns.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var docs []returnDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]dr.Request, len(docs))
	for i, d := range docs {
		out[i] = d.request()
	}
	return out, nil
}

func (r *ReturnRepository) ListByOrder(ctx context.Context, orderID string) ([]dr.Request, error) {
	id, err := oid(orderID)
	if err != nil {
		return []dr.Request{}, nil
	}
	return r.find(ctx, bson.M{"order_id": id}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
}

func (r *ReturnRepository) ListByCustomer(ctx context.Context, customerID string) ([]dr.Request, error) {
	id, err := oid(customerID)
	if err != nil {
		return []dr.Request{}, nil
	}
	return r.find(ctx, bson.M{"customer_id": id}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(200))
}

func (r *ReturnRepository) List(ctx context.Context, f returnsapp.Filter) ([]dr.Request, int64, error) {
	filter := bson.M{}
	if f.Status != "" {
		filter["status"] = f.Status
	}
	if f.OrderID != "" {
		id, err := oid(f.OrderID)
		if err != nil {
			return []dr.Request{}, 0, nil
		}
		filter["order_id"] = id
	}
	total, err := r.returns.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	items, err := r.find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetSkip((f.Page-1)*f.Size).SetLimit(f.Size))
	return items, total, err
}

func (r *ReturnRepository) Transition(ctx context.Context, id, from string, req dr.Request) (*dr.Request, error) {
	d, err := toReturnDoc(req)
	if err != nil {
		return nil, err
	}
	set := bson.M{"status": d.Status, "admin_note": d.AdminNote, "restocked": d.Restocked, "history": d.History, "updated_at": d.UpdatedAt}
	update := bson.M{"$set": set}
	if d.Open == nil {
		update["$unset"] = bson.M{"open_order_id": ""}
	}
	var out returnDoc
	err = r.returns.FindOneAndUpdate(ctx, bson.M{"_id": d.ID, "status": from}, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&out)
	if errors.Is(err, driver.ErrNoDocuments) {
		return nil, returnsapp.ErrConflict
	}
	if err != nil {
		return nil, err
	}
	result := out.request()
	return &result, nil
}

func (r *ReturnRepository) Restock(ctx context.Context, items []dr.Item) error {
	lines := make([]domainorder.OrderItem, len(items))
	for i, it := range items {
		lines[i] = domainorder.OrderItem{ProductID: it.ProductID, SKU: it.SKU, Quantity: it.Quantity}
	}
	return r.orders.incrementOrderStock(ctx, lines)
}

func (r *ReturnRepository) MarkOrderRefunded(ctx context.Context, orderID string, at time.Time) error {
	id, err := oid(orderID)
	if err != nil {
		return nil
	}
	_, err = r.orders.orders.UpdateOne(ctx,
		bson.M{"_id": id, "payment_status": domainorder.PaymentPaid},
		bson.M{"$set": bson.M{"payment_status": domainorder.PaymentRefunded, "refunded_at": at, "updated_at": at}})
	return err
}
