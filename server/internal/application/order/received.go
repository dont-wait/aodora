package order

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

// ReceiptStore marks an order delivered and paid in one conditional write.
type ReceiptStore interface {
	// ConfirmReceived moves id from status `from` to delivered, records
	// delivered_at, and marks an unpaid order paid. ErrStateConflict when the
	// order is no longer in `from`.
	ConfirmReceived(ctx context.Context, id, from string) (*domain.Order, error)
}

// ConfirmReceived is the buyer's "Đã nhận được hàng": it closes delivery,
// records payment (COD money changes hands at the door, counter orders at the
// till) and starts the return window. The shop cannot do this for the buyer.
func (s *Service) ConfirmReceived(ctx context.Context, viewer Viewer, id string) (*domain.Order, error) {
	viewer.IsAdmin = false // ownership only: an admin session is not the buyer
	o, err := s.Get(ctx, viewer, id)
	if err != nil {
		return nil, err
	}
	if o.Status == domain.StatusDelivered {
		return o, nil
	}
	if !domain.CustomerCanConfirmReceived(o.PaymentMethod, o.Status) {
		return nil, ErrStateConflict
	}
	store, ok := s.store.(ReceiptStore)
	if !ok {
		return nil, fmt.Errorf("%w: receipt confirmation not supported", ErrOrderUnavailable)
	}
	updated, err := store.ConfirmReceived(ctx, id, o.Status)
	if err != nil {
		if errors.Is(err, ErrStateConflict) || errors.Is(err, ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}
	return updated, nil
}
