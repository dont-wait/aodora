package order

import (
	"context"
	"errors"
	"testing"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

type adminFake struct {
	mutationErr error
	result      *domain.Order
	err         error
	filter      AdminFilter
	action      AdminAction
	target      string
	calls       int
}

func (f *adminFake) FindByID(context.Context, string) (*domain.Order, error) {
	f.calls++
	return f.result, f.err
}
func (f *adminFake) ListAdmin(_ context.Context, filter AdminFilter) ([]domain.Order, int64, error) {
	f.calls++
	f.filter = filter
	return nil, 0, f.err
}
func (f *adminFake) MutateAdmin(_ context.Context, _ string, a AdminAction, target string) (*domain.Order, error) {
	f.calls++
	f.action = a
	f.target = target
	return f.result, f.mutationErr
}
func TestAdminServicePortAndErrors(t *testing.T) {
	ctx := context.Background()
	id := "680000000000000000000001"
	dbErr := errors.New("database failed")
	f := &adminFake{}
	s := NewAdminService(f)
	operations := []func(string) (*domain.Order, error){func(id string) (*domain.Order, error) { return s.Get(ctx, id) }, func(id string) (*domain.Order, error) { return s.SetStatus(ctx, id, domain.StatusConfirmed) }, func(id string) (*domain.Order, error) { return s.Cancel(ctx, id) }, func(id string) (*domain.Order, error) { return s.ConfirmRestock(ctx, id) }, func(id string) (*domain.Order, error) { return s.SetPayment(ctx, id, domain.PaymentPaid) }}
	for _, op := range operations {
		for _, invalid := range []string{"", "bad", "zz0000000000000000000001"} {
			before := f.calls
			if _, err := op(invalid); !errors.Is(err, ErrNotFound) {
				t.Fatalf("bad id: %v", err)
			}
			if f.calls != before {
				t.Fatal("invalid id reached store")
			}
		}
		f.err = dbErr
		if _, err := op(id); !errors.Is(err, ErrOrderUnavailable) || !errors.Is(err, dbErr) {
			t.Fatalf("infrastructure wrapping: %v", err)
		}
	}
	f.err = nil
	if _, err := s.Get(ctx, id); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	for _, business := range []error{ErrNotFound, ErrStateConflict, ErrPaymentConflict, ErrRestockUnavailable} {
		f.err = business
		if _, err := s.Cancel(ctx, id); !errors.Is(err, business) {
			t.Fatal(err)
		}
	}
	f.err = nil
	f.result = &domain.Order{ID: id, Status: domain.StatusConfirmed, PaymentStatus: domain.PaymentUnpaid}
	for i, op := range operations {
		f.result.Status = domain.StatusPending
		if i == 3 {
			f.result.Status = domain.StatusCancelled
			f.result.InventoryRestock = domain.InventoryAwaitingReturn
		}
		got, err := op(id)
		if err != nil || got != f.result {
			t.Fatalf("port result: %v %v", got, err)
		}
	}
	if f.action != AdminSetPayment || f.target != domain.PaymentPaid {
		t.Fatal("wrong payment command")
	}
	for _, filter := range []AdminFilter{{Status: "bad"}, {PaymentStatus: "bad"}} {
		_, _, _, _, err := s.List(ctx, filter)
		var invalid *domain.InvalidFieldError
		if !errors.As(err, &invalid) {
			t.Fatal(err)
		}
	}
	_, _, page, size, err := s.List(ctx, AdminFilter{})
	if err != nil || page != 1 || size != 20 {
		t.Fatalf("defaults: %d %d %v", page, size, err)
	}
	_, _, _, size, err = s.List(ctx, AdminFilter{PageSize: 101})
	if err != nil || size != 100 || f.filter.PageSize != 100 {
		t.Fatal("clamp")
	}
	f.err = dbErr
	_, _, _, _, err = s.List(ctx, AdminFilter{})
	if !errors.Is(err, ErrOrderUnavailable) {
		t.Fatal(err)
	}
	for _, op := range []func() error{func() error { _, err := s.SetStatus(ctx, id, "unknown"); return err }, func() error { _, err := s.SetPayment(ctx, id, "unknown"); return err }} {
		var invalid *domain.InvalidFieldError
		if err := op(); !errors.As(err, &invalid) {
			t.Fatal(err)
		}
	}
}

func TestAdminServiceTransitionTables(t *testing.T) {
	ctx := context.Background()
	id := "680000000000000000000001"
	states := []string{domain.StatusPending, domain.StatusConfirmed, domain.StatusShipping, domain.StatusDelivered, domain.StatusCancelled, "unknown"}
	for _, method := range []string{"cod", "cash_at_counter", "bank_transfer"} {
		for _, from := range states {
			if f := (&adminFake{result: &domain.Order{ID: id, Status: from, PaymentMethod: method}}); true {
				// Only the buyer confirms delivery.
				if _, err := NewAdminService(f).SetStatus(ctx, id, domain.StatusDelivered); !errors.Is(err, ErrStateConflict) || f.calls != 0 {
					t.Fatalf("%s %s -> delivered by admin: %v calls=%d", method, from, err, f.calls)
				}
			}
			for _, target := range []string{"confirmed", "shipping"} {
				f := &adminFake{result: &domain.Order{ID: id, Status: from, PaymentMethod: method, PaymentStatus: domain.PaymentPaid}}
				s := NewAdminService(f)
				got, err := s.SetStatus(ctx, id, target)
				if !domain.NextStatusAllowed(method, from, target) {
					if !errors.Is(err, ErrStateConflict) || f.calls != 1 {
						t.Fatalf("%s %s -> %s: %v calls=%d", method, from, target, err, f.calls)
					}
					continue
				}
				if err != nil || got.PaymentStatus != domain.PaymentPaid {
					t.Fatalf("status changed payment: %v", err)
				}
				wantCalls := 2
				if from == target {
					wantCalls = 1
				}
				if f.calls != wantCalls {
					t.Fatal("incorrect status no-op")
				}
			}
		}
	}
	for _, from := range []string{"unpaid", "paid", "refunded", "unknown"} {
		for _, target := range []string{"paid", "refunded"} {
			f := &adminFake{result: &domain.Order{ID: id, Status: domain.StatusCancelled, PaymentStatus: from}}
			s := NewAdminService(f)
			got, err := s.SetPayment(ctx, id, target)
			if !domain.NextPaymentAllowed(from, target) {
				if !errors.Is(err, ErrPaymentConflict) || f.calls != 1 {
					t.Fatalf("payment %s -> %s: %v", from, target, err)
				}
				continue
			}
			if err != nil || got.Status != domain.StatusCancelled {
				t.Fatal("payment changed delivery")
			}
			want := 2
			if from == target {
				want = 1
			}
			if f.calls != want {
				t.Fatal("incorrect payment no-op")
			}
		}
	}
	for _, state := range states {
		f := &adminFake{result: &domain.Order{ID: id, Status: state, PaymentStatus: "paid"}}
		s := NewAdminService(f)
		got, err := s.Cancel(ctx, id)
		if state == "delivered" || state == "unknown" {
			if !errors.Is(err, ErrStateConflict) {
				t.Fatal(err)
			}
		} else {
			if err != nil || got.PaymentStatus != "paid" {
				t.Fatal("cancel changed payment")
			}
			want := 2
			if state == "cancelled" {
				want = 1
			}
			if f.calls != want {
				t.Fatal("incorrect cancel no-op")
			}
		}
		for _, marker := range []string{"", "awaiting_return", "restocked", "not_applicable"} {
			f.calls = 0
			f.result.InventoryRestock = marker
			_, err := s.ConfirmRestock(ctx, id)
			allowed := state == "cancelled" && (marker == "awaiting_return" || marker == "restocked")
			if !allowed {
				if !errors.Is(err, ErrStateConflict) {
					t.Fatal(err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				want := 2
				if marker == "restocked" {
					want = 1
				}
				if f.calls != want {
					t.Fatal("incorrect restock no-op")
				}
			}
		}
	}
}

func TestAdminDisallowedTargets(t *testing.T) {
	ctx := context.Background()
	id := "680000000000000000000001"
	f := &adminFake{}
	s := NewAdminService(f)
	for _, target := range []string{"pending", "cancelled"} {
		if _, err := s.SetStatus(ctx, id, target); !errors.Is(err, ErrStateConflict) {
			t.Fatal(err)
		}
	}
	if _, err := s.SetPayment(ctx, id, "unpaid"); !errors.Is(err, ErrPaymentConflict) {
		t.Fatal(err)
	}
	if f.calls != 0 {
		t.Fatal("invalid target reached store")
	}
}

func TestAdminMutationStoreErrors(t *testing.T) {
	ctx := context.Background()
	id := "680000000000000000000001"
	cause := errors.New("conditional update failed")
	for _, storeErr := range []error{cause, ErrNotFound, ErrStateConflict, ErrPaymentConflict, ErrRestockUnavailable} {
		f := &adminFake{result: &domain.Order{ID: id, Status: "pending"}, mutationErr: storeErr}
		s := NewAdminService(f)
		_, err := s.Cancel(ctx, id)
		if storeErr == cause {
			if !errors.Is(err, ErrOrderUnavailable) || !errors.Is(err, cause) {
				t.Fatal(err)
			}
		} else if !errors.Is(err, storeErr) {
			t.Fatal(err)
		}
		if f.calls != 2 {
			t.Fatal("mutation did not reach store")
		}
	}
}
