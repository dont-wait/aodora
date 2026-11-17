package order

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

var (
	ErrStateConflict      = errors.New("order state conflict")
	ErrPaymentConflict    = errors.New("order payment conflict")
	ErrRestockUnavailable = errors.New("order restock unavailable")
)

type AdminFilter struct {
	Status        string
	PaymentStatus string
	Page          int64
	PageSize      int64
}

type AdminAction string

const (
	AdminSetStatus      AdminAction = "status"
	AdminCancel         AdminAction = "cancel"
	AdminConfirmRestock AdminAction = "restock"
	AdminSetPayment     AdminAction = "payment"
)

// AdminStore owns atomic conditional writes and inventory transactions.
// It returns ErrNotFound for absent IDs and business sentinels for conflicts.
type AdminStore interface {
	FindByID(context.Context, string) (*domain.Order, error)
	ListAdmin(context.Context, AdminFilter) ([]domain.Order, int64, error)
	MutateAdmin(context.Context, string, AdminAction, string) (*domain.Order, error)
}

type AdminService struct{ store AdminStore }

func NewAdminService(store AdminStore) *AdminService { return &AdminService{store: store} }

func adminError(err error) error {
	if err == nil {
		return nil
	}
	for _, business := range []error{ErrNotFound, ErrStateConflict, ErrPaymentConflict, ErrRestockUnavailable} {
		if errors.Is(err, business) {
			return err
		}
	}
	return fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
}

func validAdminID(id string) bool {
	if len(id) != 24 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func (s *AdminService) Get(ctx context.Context, id string) (*domain.Order, error) {
	if !validAdminID(id) {
		return nil, ErrNotFound
	}
	o, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, adminError(err)
	}
	if o == nil {
		return nil, ErrNotFound
	}
	return o, nil
}

func (s *AdminService) List(ctx context.Context, f AdminFilter) ([]domain.Order, int64, int64, int64, error) {
	switch f.Status {
	case "", domain.StatusPending, domain.StatusConfirmed, domain.StatusShipping, domain.StatusDelivered, domain.StatusCancelled:
	default:
		return nil, 0, 0, 0, domain.NewInvalidFieldError("status", "Trạng thái đơn không hợp lệ")
	}
	switch f.PaymentStatus {
	case "", domain.PaymentUnpaid, domain.PaymentPaid, domain.PaymentRefunded:
	default:
		return nil, 0, 0, 0, domain.NewInvalidFieldError("payment_status", "Trạng thái thanh toán không hợp lệ")
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	if f.Page-1 > math.MaxInt64/f.PageSize {
		return nil, 0, 0, 0, domain.NewInvalidFieldError("page", "Trang vượt giới hạn")
	}
	orders, total, err := s.store.ListAdmin(ctx, f)
	return orders, total, f.Page, f.PageSize, adminError(err)
}

func (s *AdminService) mutate(ctx context.Context, id string, action AdminAction, target string) (*domain.Order, error) {
	if !validAdminID(id) {
		return nil, ErrNotFound
	}
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// These checks also make retries cheap. The store MUST re-evaluate against
	// the current document inside its conditional write/transaction.
	switch action {
	case AdminSetStatus:
		if !domain.NextStatusAllowed(current.PaymentMethod, current.Status, target) {
			return nil, ErrStateConflict
		}
		if current.Status == target {
			return current, nil
		}
	case AdminSetPayment:
		if !domain.NextPaymentAllowed(current.PaymentStatus, target) {
			return nil, ErrPaymentConflict
		}
		if current.PaymentStatus == target {
			return current, nil
		}
	case AdminCancel:
		if current.Status == domain.StatusCancelled {
			return current, nil
		}
		if current.Status != domain.StatusPending && current.Status != domain.StatusConfirmed && current.Status != domain.StatusShipping {
			return nil, ErrStateConflict
		}
	case AdminConfirmRestock:
		if current.Status != domain.StatusCancelled {
			return nil, ErrStateConflict
		}
		if current.InventoryRestock == domain.InventoryRestocked {
			return current, nil
		}
		if current.InventoryRestock != domain.InventoryAwaitingReturn {
			return nil, ErrStateConflict
		}
	}
	o, err := s.store.MutateAdmin(ctx, id, action, target)
	return o, adminError(err)
}

func (s *AdminService) SetStatus(ctx context.Context, id, target string) (*domain.Order, error) {
	if target == domain.StatusPending || target == domain.StatusCancelled {
		return nil, ErrStateConflict
	}
	if target != domain.StatusConfirmed && target != domain.StatusShipping && target != domain.StatusDelivered {
		return nil, domain.NewInvalidFieldError("status", "Trạng thái đích không hợp lệ")
	}
	return s.mutate(ctx, id, AdminSetStatus, target)
}

func (s *AdminService) Cancel(ctx context.Context, id string) (*domain.Order, error) {
	return s.mutate(ctx, id, AdminCancel, "")
}

func (s *AdminService) ConfirmRestock(ctx context.Context, id string) (*domain.Order, error) {
	return s.mutate(ctx, id, AdminConfirmRestock, "")
}

func (s *AdminService) SetPayment(ctx context.Context, id, target string) (*domain.Order, error) {
	if target == domain.PaymentUnpaid {
		return nil, ErrPaymentConflict
	}
	if target != domain.PaymentPaid && target != domain.PaymentRefunded {
		return nil, domain.NewInvalidFieldError("payment_status", "Trạng thái thanh toán đích không hợp lệ")
	}
	return s.mutate(ctx, id, AdminSetPayment, target)
}
