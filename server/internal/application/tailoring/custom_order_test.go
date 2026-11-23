package tailoring

import (
	"context"
	"errors"
	"testing"
	"time"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

const prodID = "680000000000000000000001"

type fakeOrderStore struct {
	existing  *domainorder.Order
	findErr   error
	createErr []error // consumed one per call
	calls     int
	drafts    []CustomOrderDraft
	raceOrder *domainorder.Order // returned with created=false on create
}

func (f *fakeOrderStore) FindCustomOrder(context.Context, string) (*domainorder.Order, error) {
	return f.existing, f.findErr
}

func (f *fakeOrderStore) CreateCustomOrder(_ context.Context, d CustomOrderDraft) (*domainorder.Order, bool, error) {
	f.calls++
	f.drafts = append(f.drafts, d)
	if len(f.createErr) > 0 {
		err := f.createErr[0]
		f.createErr = f.createErr[1:]
		if err != nil {
			return nil, false, err
		}
	}
	if f.raceOrder != nil {
		return f.raceOrder, false, nil
	}
	o := &domainorder.Order{ID: "7a0000000000000000000001", OrderNumber: d.OrderNumber, PaymentMethod: d.PaymentMethod, Total: d.Total,
		Items: []domainorder.OrderItem{{ProductID: d.ProductID, UnitPrice: d.UnitPrice, Quantity: d.Quantity}}}
	return o, true, nil
}

type fakeProducts struct {
	p   *ProductRef
	err error
}

func (f fakeProducts) FindProductByID(context.Context, string) (*ProductRef, error) {
	return f.p, f.err
}

func mustMoney(t *testing.T, s string) domainorder.Money {
	t.Helper()
	m, err := domainorder.ParseMoney(s)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func newCustomSvc(t *testing.T, status string, m *domain.Measurements, fee *domainorder.Money) (*CustomOrderService, *fakeAdminStore, *fakeOrderStore) {
	t.Helper()
	reqs := &fakeAdminStore{doc: &domain.TailoringRequest{ID: testID, FullName: "Khách A", Phone: "0900000001", Status: status, Measurements: m,
		CreatedAt: time.Unix(1000, 0).UTC(), UpdatedAt: time.Unix(1000, 0).UTC()}}
	st := &fakeOrderStore{}
	n := 0
	svc := NewCustomOrderService(reqs, fakeProducts{p: &ProductRef{ID: prodID, Name: "Áo dài", IsActive: true}}, st, fee,
		func(time.Time) string { n++; return "AD-TEST-" + string(rune('A'+n)) })
	return svc, reqs, st
}

func input() CustomOrderInput {
	p := prodID
	return CustomOrderInput{ProductID: &p, UnitPrice: "1500000.50", PaymentMethod: "cash_at_counter"}
}

func ptrMoney(t *testing.T, s string) *domainorder.Money { m := mustMoney(t, s); return &m }

func TestCustomOrderCounterSuccess(t *testing.T) {
	svc, _, st := newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	in := input()
	q := int64(2)
	in.Quantity = &q
	o, created, err := svc.Create(context.Background(), testID, in)
	if err != nil || !created || o == nil {
		t.Fatalf("create: %v %v", err, created)
	}
	d := st.drafts[0]
	if d.Subtotal.String() != "3000001" || d.ShippingFee.String() != "0" || d.Total.String() != "3000001" {
		t.Fatalf("money exact: %s %s %s", d.Subtotal, d.ShippingFee, d.Total)
	}
	if d.Address.Recipient != "Khách A" || d.Address.Phone != "0900000001" || d.Address.Line1 != "" {
		t.Fatalf("address defaults: %+v", d.Address)
	}
	if d.ProductName != "Áo dài" || d.Quantity != 2 {
		t.Fatalf("draft: %+v", d)
	}
}

func TestCustomOrderCODFeeAndAddress(t *testing.T) {
	// no fee configured => unavailable, nothing created
	svc, _, st := newCustomSvc(t, domain.StatusInProgress, fullMeasurements(), nil)
	in := input()
	in.PaymentMethod = "cod"
	in.Address.Line1, in.Address.City = "1 Lê Lợi", "Huế"
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.Is(err, ErrShippingUnavailable) || st.calls != 0 {
		t.Fatalf("expected ErrShippingUnavailable, got %v", err)
	}
	// fee configured => added exactly
	svc, _, st = newCustomSvc(t, domain.StatusInProgress, fullMeasurements(), ptrMoney(t, "30000"))
	if _, _, err := svc.Create(context.Background(), testID, in); err != nil {
		t.Fatal(err)
	}
	if st.drafts[0].Total.String() != "1530000.5" || st.drafts[0].ShippingFee.String() != "30000" {
		t.Fatalf("total=%s fee=%s", st.drafts[0].Total, st.drafts[0].ShippingFee)
	}
	// missing COD address => 400, no fake address
	in.Address.Line1 = ""
	var ife *domain.InvalidFieldError
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.As(err, &ife) || ife.Field != "shipping_address.line1" {
		t.Fatalf("expected line1 error, got %v", err)
	}
}

func TestCustomOrderValidation(t *testing.T) {
	svc, _, st := newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	var ife *domain.InvalidFieldError
	for name, mutate := range map[string]func(*CustomOrderInput){
		"price empty":    func(i *CustomOrderInput) { i.UnitPrice = "" },
		"price zero":     func(i *CustomOrderInput) { i.UnitPrice = "0" },
		"price zero dec": func(i *CustomOrderInput) { i.UnitPrice = "0.00" },
		"price negative": func(i *CustomOrderInput) { i.UnitPrice = "-5" },
		"price text":     func(i *CustomOrderInput) { i.UnitPrice = "abc" },
		"price exp":      func(i *CustomOrderInput) { i.UnitPrice = "1e6" },
		"qty zero":       func(i *CustomOrderInput) { q := int64(0); i.Quantity = &q },
		"qty huge":       func(i *CustomOrderInput) { q := int64(2000000); i.Quantity = &q },
		"method":         func(i *CustomOrderInput) { i.PaymentMethod = "bank_transfer" },
		"bad product id": func(i *CustomOrderInput) { b := "zz"; i.ProductID = &b },
	} {
		in := input()
		mutate(&in)
		if _, _, err := svc.Create(context.Background(), testID, in); !errors.As(err, &ife) {
			t.Fatalf("%s: expected invalid field, got %v", name, err)
		}
	}
	if st.calls != 0 {
		t.Fatal("invalid input reached store")
	}
}

func TestCustomOrderStateAndMeasurements(t *testing.T) {
	for _, s := range []string{domain.StatusSubmitted, domain.StatusContacted, domain.StatusConsultationScheduled, domain.StatusCompleted, domain.StatusCancelled} {
		svc, _, st := newCustomSvc(t, s, fullMeasurements(), nil)
		if _, _, err := svc.Create(context.Background(), testID, input()); !errors.Is(err, ErrStateConflict) || st.calls != 0 {
			t.Fatalf("%s: expected state conflict, got %v", s, err)
		}
	}
	m := fullMeasurements()
	m.HipsCM = nil
	svc, _, _ := newCustomSvc(t, domain.StatusMeasurementConfirmed, m, nil)
	if _, _, err := svc.Create(context.Background(), testID, input()); !errors.Is(err, ErrMeasurementIncomplete) {
		t.Fatalf("expected incomplete, got %v", err)
	}
	if _, _, err := svc.Create(context.Background(), "bad", input()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestCustomOrderProductRules(t *testing.T) {
	svc, reqs, _ := newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	var ife *domain.InvalidFieldError
	// no product anywhere
	in := input()
	in.ProductID = nil
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.As(err, &ife) || ife.Field != "product_id" {
		t.Fatalf("expected product_id required, got %v", err)
	}
	// request has product A, caller sends B
	a := "680000000000000000000009"
	reqs.doc.ProductID = &a
	if _, _, err := svc.Create(context.Background(), testID, input()); !errors.As(err, &ife) || ife.Field != "product_id" {
		t.Fatalf("expected mismatch, got %v", err)
	}
	// request has product, caller omits: uses request's
	in.ProductID = nil
	if _, _, err := svc.Create(context.Background(), testID, in); err != nil {
		t.Fatalf("should use request product: %v", err)
	}
	// inactive / missing / infra
	svc.products = fakeProducts{p: &ProductRef{ID: a, IsActive: false}}
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.As(err, &ife) {
		t.Fatalf("inactive: %v", err)
	}
	svc.products = fakeProducts{}
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.As(err, &ife) {
		t.Fatalf("missing: %v", err)
	}
	svc.products = fakeProducts{err: errors.New("down")}
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("infra: %v", err)
	}
}

func TestCustomOrderReplayAndConflict(t *testing.T) {
	svc, reqs, st := newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	st.existing = &domainorder.Order{ID: "7a0000000000000000000001", PaymentMethod: "cash_at_counter",
		Items: []domainorder.OrderItem{{ProductID: prodID, UnitPrice: mustMoney(t, "1500000.5"), Quantity: 1}}}
	// identical retry => replay, even after the request moved on
	reqs.doc.Status = domain.StatusCompleted
	o, created, err := svc.Create(context.Background(), testID, input())
	if err != nil || created || o.ID != st.existing.ID || st.calls != 0 {
		t.Fatalf("replay: %v created=%v calls=%d", err, created, st.calls)
	}
	// different price => conflict, never a second order
	in := input()
	in.UnitPrice = "1"
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.Is(err, ErrCustomOrderExists) {
		t.Fatalf("expected exists, got %v", err)
	}
	in = input()
	in.PaymentMethod = "cod"
	in.Address.Line1, in.Address.City = "x", "y"
	if _, _, err := svc.Create(context.Background(), testID, in); !errors.Is(err, ErrCustomOrderExists) {
		t.Fatalf("expected exists for different method, got %v", err)
	}
	if st.calls != 0 {
		t.Fatal("must not create")
	}
}

func TestCustomOrderRaceAndRetries(t *testing.T) {
	svc, _, st := newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	// store reports another request won: same content => replayed (created=false)
	st.raceOrder = &domainorder.Order{ID: "7a0000000000000000000002", PaymentMethod: "cash_at_counter",
		Items: []domainorder.OrderItem{{ProductID: prodID, UnitPrice: mustMoney(t, "1500000.5"), Quantity: 1}}}
	o, created, err := svc.Create(context.Background(), testID, input())
	if err != nil || created || o.ID != "7a0000000000000000000002" {
		t.Fatalf("race replay: %v %v", err, created)
	}
	// duplicate order number => retried with a new number, then succeeds
	svc, _, st = newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	st.createErr = []error{ErrDuplicateOrderNumber, ErrDuplicateOrderNumber}
	if _, created, err := svc.Create(context.Background(), testID, input()); err != nil || !created || st.calls != 3 {
		t.Fatalf("dup retries: %v %v calls=%d", err, created, st.calls)
	}
	if st.drafts[0].OrderNumber == st.drafts[1].OrderNumber {
		t.Fatal("order number must change between attempts")
	}
	// exhausted => unavailable
	svc, _, st = newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	st.createErr = []error{ErrDuplicateOrderNumber, ErrDuplicateOrderNumber, ErrDuplicateOrderNumber}
	if _, _, err := svc.Create(context.Background(), testID, input()); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("exhausted: %v", err)
	}
	// infra error => unavailable, not a 400
	svc, _, st = newCustomSvc(t, domain.StatusMeasurementConfirmed, fullMeasurements(), nil)
	st.createErr = []error{errors.New("tx failed")}
	if _, _, err := svc.Create(context.Background(), testID, input()); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("infra: %v", err)
	}
}
