package returns

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	dr "github.com/your-org/aodora/server/internal/domain/returns"
)

const prefix = "http://cdn/uploads/returns/"

type fakeOrders struct{ o *domainorder.Order }

func (f *fakeOrders) FindByID(_ context.Context, id string) (*domainorder.Order, error) {
	if f.o == nil || f.o.ID != id {
		return nil, nil
	}
	c := *f.o
	return &c, nil
}

type fakeStore struct {
	reqs     map[string]dr.Request
	restock  []dr.Item
	refunded string
	n        int
}

func (f *fakeStore) Create(_ context.Context, r dr.Request) (dr.Request, error) {
	for _, e := range f.reqs {
		if e.OrderID == r.OrderID && dr.Open(e.Status) {
			return dr.Request{}, ErrConflict
		}
	}
	f.n++
	r.ID = string(rune('a' + f.n))
	f.reqs[r.ID] = r
	return r, nil
}
func (f *fakeStore) Get(_ context.Context, id string) (*dr.Request, error) {
	r, ok := f.reqs[id]
	if !ok {
		return nil, nil
	}
	return &r, nil
}
func (f *fakeStore) ListByOrder(_ context.Context, id string) ([]dr.Request, error) {
	var out []dr.Request
	for _, r := range f.reqs {
		if r.OrderID == id {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeStore) ListByCustomer(context.Context, string) ([]dr.Request, error) { return nil, nil }
func (f *fakeStore) List(context.Context, Filter) ([]dr.Request, int64, error)    { return nil, 0, nil }
func (f *fakeStore) Transition(_ context.Context, id, from string, r dr.Request) (*dr.Request, error) {
	if f.reqs[id].Status != from {
		return nil, ErrConflict
	}
	f.reqs[id] = r
	return &r, nil
}
func (f *fakeStore) Restock(_ context.Context, items []dr.Item) error { f.restock = items; return nil }
func (f *fakeStore) MarkOrderRefunded(_ context.Context, id string, _ time.Time) error {
	f.refunded = id
	return nil
}

var delivered = time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)

func setup(t *testing.T, now time.Time) (*Service, *fakeStore) {
	t.Helper()
	price, _ := domainorder.ParseMoney("1200000")
	o := &domainorder.Order{ID: "o1", OrderNumber: "AD-1", CustomerID: "c1", Status: domainorder.StatusDelivered, PaymentStatus: domainorder.PaymentPaid, DeliveredAt: &delivered,
		Items: []domainorder.OrderItem{{ProductID: "p1", ProductName: "Áo dài lụa", SKU: "S1", Size: "M", Quantity: 2, UnitPrice: price}}}
	store := &fakeStore{reqs: map[string]dr.Request{}}
	s := NewService(store, &fakeOrders{o: o}, func(id string) string { return prefix + id + "/" })
	s.now = func() time.Time { return now }
	return s, store
}

func validInput() Input {
	return Input{OrderID: "o1", Items: []ItemInput{{SKU: "S1", Quantity: 1}}, Reason: dr.ReasonDefect, Evidence: []EvidenceInput{{URL: prefix + "c1/2026/11/x.jpg"}}}
}

func TestCreateWithinThreeDays(t *testing.T) {
	s, _ := setup(t, delivered.Add(71*time.Hour))
	r, err := s.Create(context.Background(), "c1", validInput())
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != dr.StatusRequested || r.RefundAmount != 1200000 || r.Evidence[0].Kind != dr.EvidenceImage {
		t.Fatalf("%+v", r)
	}
	if _, err := s.Create(context.Background(), "c1", validInput()); !errors.As(err, new(NotEligible)) {
		t.Fatalf("second open request: %v", err)
	}
}

func TestCreateRefusedAfterWindow(t *testing.T) {
	s, _ := setup(t, delivered.Add(72*time.Hour+time.Minute))
	_, err := s.Create(context.Background(), "c1", validInput())
	var ne NotEligible
	if !errors.As(err, &ne) || !strings.Contains(ne.Message, "3 ngày") {
		t.Fatalf("%v", err)
	}
	e, _ := s.Eligibility(context.Background(), "c1", "o1")
	if e.Eligible || e.Deadline == nil {
		t.Fatalf("%+v", e)
	}
}

func TestCreateValidation(t *testing.T) {
	s, _ := setup(t, delivered.Add(time.Hour))
	cases := map[string]func(*Input){
		"evidence":          func(in *Input) { in.Evidence = nil },
		"evidence[0]":       func(in *Input) { in.Evidence = []EvidenceInput{{URL: "http://evil/x.jpg"}} },
		"items[0].quantity": func(in *Input) { in.Items[0].Quantity = 3 },
		"items[0]":          func(in *Input) { in.Items[0].SKU = "nope" },
		"reason":            func(in *Input) { in.Reason = "x" },
		"note":              func(in *Input) { in.Reason = dr.ReasonOther },
	}
	for field, mutate := range cases {
		in := validInput()
		mutate(&in)
		_, err := s.Create(context.Background(), "c1", in)
		var inv Invalid
		if !errors.As(err, &inv) || inv.Field != field {
			t.Errorf("%s: %v", field, err)
		}
	}
	// someone else's evidence prefix
	in := validInput()
	in.Evidence = []EvidenceInput{{URL: prefix + "c2/2026/11/x.jpg"}}
	if _, err := s.Create(context.Background(), "c1", in); err == nil {
		t.Error("evidence of another customer accepted")
	}
	if _, err := s.Create(context.Background(), "c2", validInput()); !errors.Is(err, ErrNotFound) {
		t.Errorf("other customer's order: %v", err)
	}
}

func TestAdminLifecycleRestockAndRefund(t *testing.T) {
	s, store := setup(t, delivered.Add(time.Hour))
	in := validInput()
	in.Items[0].Quantity = 2
	r, err := s.Create(context.Background(), "c1", in)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.AdminTransition(ctx, r.ID, TransitionInput{To: dr.StatusRefunded}); !errors.Is(err, ErrConflict) {
		t.Fatalf("skip to refunded: %v", err)
	}
	for _, step := range []TransitionInput{{To: dr.StatusApproved}, {To: dr.StatusCollecting}, {To: dr.StatusCollected, Restock: true}, {To: dr.StatusRefunded}} {
		if _, err := s.AdminTransition(ctx, r.ID, step); err != nil {
			t.Fatalf("%s: %v", step.To, err)
		}
	}
	got := store.reqs[r.ID]
	if got.Status != dr.StatusRefunded || !got.Restocked || len(got.History) != 5 || len(store.restock) != 1 || store.refunded != "o1" {
		t.Fatalf("%+v restock=%v refunded=%s", got, store.restock, store.refunded)
	}
}

func TestRejectNeedsReasonAndCustomerCancel(t *testing.T) {
	s, store := setup(t, delivered.Add(time.Hour))
	r, _ := s.Create(context.Background(), "c1", validInput())
	if _, err := s.AdminTransition(context.Background(), r.ID, TransitionInput{To: dr.StatusRejected}); !errors.As(err, new(Invalid)) {
		t.Fatalf("reject without note: %v", err)
	}
	if _, err := s.Cancel(context.Background(), "c1", r.ID); err != nil {
		t.Fatal(err)
	}
	if store.reqs[r.ID].Status != dr.StatusCancelled {
		t.Fatal("not cancelled")
	}
	if _, err := s.Create(context.Background(), "c1", validInput()); err != nil {
		t.Fatalf("may ask again after cancelling: %v", err)
	}
}

func TestPartialReturnDoesNotRefundWholeOrder(t *testing.T) {
	s, store := setup(t, delivered.Add(time.Hour))
	r, _ := s.Create(context.Background(), "c1", validInput()) // 1 of 2
	for _, to := range []string{dr.StatusApproved, dr.StatusCollecting, dr.StatusCollected, dr.StatusRefunded} {
		if _, err := s.AdminTransition(context.Background(), r.ID, TransitionInput{To: to}); err != nil {
			t.Fatal(err)
		}
	}
	if store.refunded != "" || store.restock != nil {
		t.Fatalf("refunded=%s restock=%v", store.refunded, store.restock)
	}
}
