// Package returns lets customers ask to return delivered items (with photo or
// video evidence) and lets the shop review, collect, restock and refund them.
package returns

import (
	"context"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	dr "github.com/your-org/aodora/server/internal/domain/returns"
)

var (
	ErrNotFound = errors.New("return request not found")
	// ErrConflict: the request changed meanwhile or another open request exists.
	ErrConflict = errors.New("return request conflict")
)

// Invalid is a user-correctable input problem.
type Invalid struct{ Field, Message string }

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

// NotEligible explains why this order cannot be returned now.
type NotEligible struct{ Message string }

func (e NotEligible) Error() string { return e.Message }

type Orders interface {
	FindByID(ctx context.Context, id string) (*domainorder.Order, error)
}

type Filter struct {
	Status, OrderID string
	Page, Size      int64
}

type Store interface {
	Create(ctx context.Context, r dr.Request) (dr.Request, error) // ErrConflict when an open request exists
	Get(ctx context.Context, id string) (*dr.Request, error)
	ListByOrder(ctx context.Context, orderID string) ([]dr.Request, error)
	ListByCustomer(ctx context.Context, customerID string) ([]dr.Request, error)
	List(ctx context.Context, f Filter) ([]dr.Request, int64, error)
	// Transition moves id from `from` to r.Status atomically (ErrConflict if it is no longer `from`).
	Transition(ctx context.Context, id, from string, r dr.Request) (*dr.Request, error)
	Restock(ctx context.Context, items []dr.Item) error
	MarkOrderRefunded(ctx context.Context, orderID string, at time.Time) error
}

type Service struct {
	store          Store
	orders         Orders
	evidencePrefix func(customerID string) string
	now            func() time.Time
}

func NewService(store Store, orders Orders, evidencePrefix func(customerID string) string) *Service {
	return &Service{store: store, orders: orders, evidencePrefix: evidencePrefix, now: time.Now}
}

type Eligibility struct {
	Eligible bool         `json:"eligible"`
	Reason   string       `json:"reason,omitempty"`
	Deadline *time.Time   `json:"deadline,omitempty"`
	Requests []dr.Request `json:"requests"`
}

type ItemInput struct {
	SKU      string `json:"sku"`
	Quantity int64  `json:"quantity"`
}

type EvidenceInput struct {
	URL string `json:"url"`
}

type Input struct {
	OrderID  string          `json:"order_id"`
	Items    []ItemInput     `json:"items"`
	Reason   string          `json:"reason"`
	Note     string          `json:"note"`
	Evidence []EvidenceInput `json:"evidence"`
}

func deliveredAt(o *domainorder.Order) time.Time {
	if o.DeliveredAt != nil {
		return *o.DeliveredAt
	}
	// Orders delivered before delivered_at existed: the delivery was their last change.
	return o.UpdatedAt
}

func (s *Service) ownOrder(ctx context.Context, customerID, orderID string) (*domainorder.Order, error) {
	if customerID == "" || orderID == "" {
		return nil, ErrNotFound
	}
	o, err := s.orders.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if o == nil || o.CustomerID != customerID {
		return nil, ErrNotFound
	}
	return o, nil
}

// Eligibility tells the order page whether to offer "Yêu cầu hoàn hàng".
func (s *Service) Eligibility(ctx context.Context, customerID, orderID string) (Eligibility, error) {
	o, err := s.ownOrder(ctx, customerID, orderID)
	if err != nil {
		return Eligibility{}, err
	}
	existing, err := s.store.ListByOrder(ctx, orderID)
	if err != nil {
		return Eligibility{}, err
	}
	out := Eligibility{Requests: existing}
	if o.Status == domainorder.StatusDelivered {
		d := dr.Deadline(deliveredAt(o))
		out.Deadline = &d
	}
	if reason := s.blocker(o, existing); reason != "" {
		out.Reason = reason
		return out, nil
	}
	out.Eligible = true
	return out, nil
}

func (s *Service) blocker(o *domainorder.Order, existing []dr.Request) string {
	if o.Status != domainorder.StatusDelivered {
		return "Chỉ gửi yêu cầu hoàn hàng sau khi đã nhận áo."
	}
	for _, r := range existing {
		if dr.Open(r.Status) {
			return "Đơn này đang có yêu cầu hoàn hàng."
		}
		if r.Status == dr.StatusRefunded {
			return "Đơn này đã được hoàn tiền."
		}
	}
	if s.now().After(dr.Deadline(deliveredAt(o))) {
		return "Đã quá 3 ngày kể từ khi nhận áo nên không thể gửi yêu cầu hoàn hàng."
	}
	return ""
}

func evidenceKind(url string) string {
	switch strings.ToLower(path.Ext(url)) {
	case ".jpg", ".jpeg", ".png", ".webp":
		return dr.EvidenceImage
	case ".mp4", ".mov", ".webm":
		return dr.EvidenceVideo
	}
	return ""
}

var contentTypes = map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".mp4": "video/mp4", ".mov": "video/quicktime", ".webm": "video/webm"}

func (s *Service) Create(ctx context.Context, customerID string, in Input) (dr.Request, error) {
	o, err := s.ownOrder(ctx, customerID, strings.TrimSpace(in.OrderID))
	if err != nil {
		return dr.Request{}, err
	}
	existing, err := s.store.ListByOrder(ctx, o.ID)
	if err != nil {
		return dr.Request{}, err
	}
	if reason := s.blocker(o, existing); reason != "" {
		return dr.Request{}, NotEligible{reason}
	}
	if len(in.Items) == 0 {
		return dr.Request{}, Invalid{"items", "Chọn ít nhất một áo cần hoàn."}
	}
	ordered := map[string]domainorder.OrderItem{}
	for _, it := range o.Items {
		ordered[it.SKU] = it
	}
	var items []dr.Item
	var refund int64
	seen := map[string]bool{}
	for i, it := range in.Items {
		field := fmt.Sprintf("items[%d]", i)
		line, ok := ordered[it.SKU]
		if !ok || seen[it.SKU] {
			return dr.Request{}, Invalid{field, "Áo này không có trong đơn hoặc bị chọn trùng."}
		}
		seen[it.SKU] = true
		if it.Quantity < 1 || it.Quantity > line.Quantity {
			return dr.Request{}, Invalid{field + ".quantity", fmt.Sprintf("Số lượng hoàn phải từ 1 đến %d.", line.Quantity)}
		}
		price, ok := line.UnitPrice.Int64()
		if !ok {
			return dr.Request{}, Invalid{field, "Không tính được giá hoàn của áo này."}
		}
		items = append(items, dr.Item{ProductID: line.ProductID, ProductName: line.ProductName, SKU: line.SKU, Size: line.Size, Color: line.Color, Quantity: it.Quantity, UnitPrice: price})
		refund += price * it.Quantity
	}
	if !dr.ValidReason(in.Reason) {
		return dr.Request{}, Invalid{"reason", "Chọn lý do hoàn hàng."}
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 1000 {
		return dr.Request{}, Invalid{"note", "Mô tả tối đa 1000 ký tự."}
	}
	if in.Reason == dr.ReasonOther && utf8.RuneCountInString(note) < 10 {
		return dr.Request{}, Invalid{"note", "Hãy mô tả lý do (ít nhất 10 ký tự)."}
	}
	if len(in.Evidence) == 0 {
		return dr.Request{}, Invalid{"evidence", "Cần ít nhất 1 ảnh hoặc 1 video minh chứng."}
	}
	if len(in.Evidence) > dr.MaxEvidence {
		return dr.Request{}, Invalid{"evidence", fmt.Sprintf("Tối đa %d ảnh/video.", dr.MaxEvidence)}
	}
	prefix := s.evidencePrefix(customerID)
	var evidence []dr.Evidence
	videos := 0
	dup := map[string]bool{}
	for i, e := range in.Evidence {
		url := strings.TrimSpace(e.URL)
		kind := evidenceKind(url)
		// Only files this customer uploaded through the evidence endpoint.
		if prefix == "" || !strings.HasPrefix(url, prefix) || strings.Contains(url[len(prefix):], "..") || kind == "" || dup[url] {
			return dr.Request{}, Invalid{fmt.Sprintf("evidence[%d]", i), "Minh chứng không hợp lệ, hãy tải lại file."}
		}
		dup[url] = true
		if kind == dr.EvidenceVideo {
			videos++
		}
		evidence = append(evidence, dr.Evidence{URL: url, Kind: kind, ContentType: contentTypes[strings.ToLower(path.Ext(url))]})
	}
	if videos > 2 {
		return dr.Request{}, Invalid{"evidence", "Tối đa 2 video."}
	}
	now := s.now().UTC()
	req := dr.Request{
		OrderID: o.ID, OrderNumber: o.OrderNumber, CustomerID: customerID,
		CustomerName: o.ShippingAddress.Recipient, Phone: o.ShippingAddress.Phone,
		Items: items, Reason: in.Reason, Note: note, Evidence: evidence,
		Status: dr.StatusRequested, RefundAmount: refund,
		History:     []dr.Event{{Status: dr.StatusRequested, At: now, By: "customer", Note: note}},
		DeliveredAt: deliveredAt(o).UTC(), CreatedAt: now, UpdatedAt: now,
	}
	return s.store.Create(ctx, req)
}

func (s *Service) ListOwn(ctx context.Context, customerID string) ([]dr.Request, error) {
	return s.store.ListByCustomer(ctx, customerID)
}

func (s *Service) Cancel(ctx context.Context, customerID, id string) (*dr.Request, error) {
	r, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if r == nil || r.CustomerID != customerID {
		return nil, ErrNotFound
	}
	if !dr.CustomerCanCancel(r.Status) {
		return nil, NotEligible{"Cửa hàng đã xử lý yêu cầu này nên không thể huỷ."}
	}
	from := r.Status
	now := s.now().UTC()
	r.Status, r.UpdatedAt = dr.StatusCancelled, now
	r.History = append(r.History, dr.Event{Status: dr.StatusCancelled, At: now, By: "customer"})
	return s.store.Transition(ctx, id, from, *r)
}

func (s *Service) AdminList(ctx context.Context, f Filter) ([]dr.Request, int64, error) {
	if f.Status != "" && !validStatus(f.Status) {
		return nil, 0, Invalid{"status", "Trạng thái không hợp lệ."}
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Size < 1 || f.Size > 100 {
		f.Size = 20
	}
	return s.store.List(ctx, f)
}

func (s *Service) AdminGet(ctx context.Context, id string) (*dr.Request, error) {
	r, err := s.store.Get(ctx, id)
	if err == nil && r == nil {
		return nil, ErrNotFound
	}
	return r, err
}

type TransitionInput struct {
	To      string `json:"to"`
	Note    string `json:"note"`
	Restock bool   `json:"restock"`
}

func (s *Service) AdminTransition(ctx context.Context, id string, in TransitionInput) (*dr.Request, error) {
	r, err := s.AdminGet(ctx, id)
	if err != nil {
		return nil, err
	}
	if !dr.AdminCan(r.Status, in.To) {
		return nil, ErrConflict
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 1000 {
		return nil, Invalid{"note", "Ghi chú tối đa 1000 ký tự."}
	}
	if in.To == dr.StatusRejected && utf8.RuneCountInString(note) < 5 {
		return nil, Invalid{"note", "Ghi rõ lý do từ chối để khách hiểu."}
	}
	from := r.Status
	now := s.now().UTC()
	r.Status, r.UpdatedAt = in.To, now
	if note != "" {
		r.AdminNote = note
	}
	r.History = append(r.History, dr.Event{Status: in.To, At: now, By: "admin", Note: note})
	if in.To == dr.StatusCollected && in.Restock {
		r.Restocked = true
	}
	updated, err := s.store.Transition(ctx, id, from, *r)
	if err != nil {
		return nil, err
	}
	if in.To == dr.StatusCollected && in.Restock {
		if err := s.store.Restock(ctx, updated.Items); err != nil {
			return nil, err
		}
	}
	if in.To == dr.StatusRefunded {
		if err := s.refundOrderIfFull(ctx, updated, now); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// refundOrderIfFull marks the whole order refunded when every item came back.
func (s *Service) refundOrderIfFull(ctx context.Context, r *dr.Request, at time.Time) error {
	o, err := s.orders.FindByID(ctx, r.OrderID)
	if err != nil || o == nil || o.PaymentStatus != domainorder.PaymentPaid {
		return err
	}
	returned := map[string]int64{}
	for _, it := range r.Items {
		returned[it.SKU] += it.Quantity
	}
	for _, it := range o.Items {
		if returned[it.SKU] < it.Quantity {
			return nil
		}
	}
	return s.store.MarkOrderRefunded(ctx, o.ID, at)
}

func validStatus(s string) bool { return slices.Contains(dr.Statuses, s) }
