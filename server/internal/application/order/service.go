package order

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

var (
	ErrNotFound             = errors.New("order not found")
	ErrIdempotencyConflict  = errors.New("idempotency conflict: request payload differs from previous attempt")
	ErrOutOfStock           = errors.New("one or more items are out of stock")
	ErrQuoteChanged         = errors.New("quote choices or prices have changed")
	ErrQuoteExpired         = errors.New("quote token has expired")
	ErrShippingUnavailable  = errors.New("shipping method is unavailable")
	ErrOrderUnavailable     = errors.New("order service is temporarily unavailable")
	ErrStockInconsistent    = errors.New("stock is inconsistent")
	ErrDuplicateAttempt     = errors.New("duplicate checkout attempt")
	ErrDuplicateOrderNumber = errors.New("duplicate order number")
)

const orderNumberAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func DefaultOrderNumber(now time.Time) string {
	datePart := now.UTC().Format("060102")
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("AD-%s-%08d", datePart, now.UnixNano()%100000000)
	}
	for i := 0; i < 8; i++ {
		buf[i] = orderNumberAlphabet[buf[i]&0x1F]
	}
	return fmt.Sprintf("AD-%s-%s", datePart, string(buf))
}

type Principal struct {
	CustomerID string
	GuestHash  string
}

type Viewer struct {
	CustomerID string
	IsAdmin    bool
	GuestHash  string
}

type QuoteItemResult struct {
	ProductID string       `json:"product_id"`
	SKU       string       `json:"sku"`
	Quantity  int64        `json:"quantity"`
	UnitPrice domain.Money `json:"unit_price"`
}

type QuoteResult struct {
	QuoteToken  string            `json:"quote_token"`
	ExpiresAt   time.Time         `json:"expires_at"`
	Subtotal    domain.Money      `json:"subtotal"`
	ShippingFee domain.Money      `json:"shipping_fee"`
	Total       domain.Money      `json:"total"`
	Items       []QuoteItemResult `json:"items"`
}

type Service struct {
	catalog        Catalog
	store          Store
	signer         *domain.QuoteSigner
	quoteTTL       time.Duration
	codShippingFee *domain.Money
	now            func() time.Time
	newOrderNumber func() string
}

func NewService(
	catalog Catalog,
	store Store,
	signer *domain.QuoteSigner,
	quoteTTL time.Duration,
	codShippingFee *domain.Money,
	now func() time.Time,
	newOrderNumber func() string,
) *Service {
	if now == nil {
		now = time.Now
	}
	if newOrderNumber == nil {
		newOrderNumber = func() string {
			return DefaultOrderNumber(now())
		}
	}
	return &Service{
		catalog:        catalog,
		store:          store,
		signer:         signer,
		quoteTTL:       quoteTTL,
		codShippingFee: codShippingFee,
		now:            now,
		newOrderNumber: newOrderNumber,
	}
}

// Quote calculates prices and fees and returns a signed quote token without modifying database state.
func (s *Service) Quote(ctx context.Context, input domain.CheckoutInput) (*QuoteResult, error) {
	norm, err := domain.Normalize(input)
	if err != nil {
		return nil, err
	}

	if s.signer == nil {
		return nil, ErrOrderUnavailable
	}

	productIDs := make([]string, 0, len(norm.Items))
	seenProd := make(map[string]bool)
	for _, it := range norm.Items {
		if !seenProd[it.ProductID] {
			seenProd[it.ProductID] = true
			productIDs = append(productIDs, it.ProductID)
		}
	}

	products, err := s.catalog.SellableProducts(ctx, productIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}

	prodQuantities := make(map[string]int64)

	for _, it := range norm.Items {
		prod, ok := products[it.ProductID]
		if !ok {
			return nil, domain.NewInvalidFieldError(fmt.Sprintf("items[%d].product_id", it.SourceIndex), "Sản phẩm không tồn tại hoặc đã ngừng bán")
		}

		if len(prod.Variants) > 0 {
			var matchingVar *Variant
			for _, v := range prod.Variants {
				if v.SKU == it.SKU {
					matchingVar = &v
					break
				}
			}
			if matchingVar == nil {
				return nil, domain.NewInvalidFieldError(fmt.Sprintf("items[%d].sku", it.SourceIndex), "Mã SKU không hợp lệ cho sản phẩm này")
			}
			if matchingVar.Stock < it.Quantity {
				return nil, ErrOutOfStock
			}
		} else {
			if prod.SKU == "" || prod.SKU != it.SKU {
				return nil, domain.NewInvalidFieldError(fmt.Sprintf("items[%d].sku", it.SourceIndex), "Mã SKU không hợp lệ cho sản phẩm này")
			}
			if prod.Stock < it.Quantity {
				return nil, ErrOutOfStock
			}
		}

		prodQuantities[it.ProductID] += it.Quantity
	}

	for prodID, totalQty := range prodQuantities {
		if products[prodID].Stock < totalQty {
			return nil, ErrOutOfStock
		}
	}

	var shippingFee domain.Money
	if norm.PaymentMethod == domain.PaymentMethodCashAtCounter {
		shippingFee = domain.ZeroMoney
	} else {
		if s.codShippingFee == nil {
			return nil, ErrShippingUnavailable
		}
		shippingFee = *s.codShippingFee
	}

	subtotal := domain.ZeroMoney
	quoteItems := make([]domain.QuoteItem, len(norm.Items))
	itemResults := make([]QuoteItemResult, len(norm.Items))

	for i, it := range norm.Items {
		prod := products[it.ProductID]
		lineTotal, err := prod.Price.MulQty(it.Quantity)
		if err != nil {
			return nil, domain.NewInvalidFieldError("items", "Tổng tiền hàng vượt quá giới hạn tính toán")
		}
		subtotal, err = subtotal.Add(lineTotal)
		if err != nil {
			return nil, domain.NewInvalidFieldError("items", "Tổng tiền hàng vượt quá giới hạn tính toán")
		}

		quoteItems[i] = domain.QuoteItem{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Quantity:  it.Quantity,
			UnitPrice: prod.Price.String(),
		}
		itemResults[i] = QuoteItemResult{
			ProductID: it.ProductID,
			SKU:       it.SKU,
			Quantity:  it.Quantity,
			UnitPrice: prod.Price,
		}
	}

	total, err := subtotal.Add(shippingFee)
	if err != nil {
		return nil, domain.NewInvalidFieldError("items", "Tổng thanh toán vượt quá giới hạn tính toán")
	}

	now := s.now().UTC()
	expiresAt := now.Add(s.quoteTTL).Truncate(time.Second)

	payload := domain.QuotePayload{
		V:             1,
		Exp:           expiresAt.Unix(),
		PaymentMethod: norm.PaymentMethod,
		Items:         quoteItems,
		ShippingFee:   shippingFee.String(),
		Subtotal:      subtotal.String(),
		Total:         total.String(),
	}

	token, err := s.signer.Sign(payload)
	if err != nil {
		return nil, err
	}

	return &QuoteResult{
		QuoteToken:  token,
		ExpiresAt:   expiresAt,
		Subtotal:    subtotal,
		ShippingFee: shippingFee,
		Total:       total,
		Items:       itemResults,
	}, nil
}

// Create places an order idempotently inside a store transaction.
func (s *Service) Create(ctx context.Context, principal Principal, requestID string, input domain.CheckoutInput) (domain.Order, bool, error) {
	hasCustomer := principal.CustomerID != ""
	hasGuest := principal.GuestHash != ""
	if (hasCustomer && hasGuest) || (!hasCustomer && !hasGuest) {
		return domain.Order{}, false, errors.New("invalid principal: exactly one of CustomerID or GuestHash must be non-empty")
	}

	// 1. Normalize input, validate idempotency key, compute scope and request hash
	norm, err := domain.Normalize(input)
	if err != nil {
		return domain.Order{}, false, err
	}

	if norm.QuoteToken == "" {
		return domain.Order{}, false, domain.NewInvalidFieldError("quote_token", "Mã quote không được để trống")
	}

	reqID, err := domain.ValidateIdempotencyKey(requestID)
	if err != nil {
		return domain.Order{}, false, err
	}

	var scope string
	if hasCustomer {
		scope, err = domain.ValidateCustomerScope(principal.CustomerID)
		if err != nil {
			return domain.Order{}, false, err
		}
	} else {
		scope = "guest:" + principal.GuestHash
	}

	canonBytes, err := domain.CanonicalJSON(norm)
	if err != nil {
		return domain.Order{}, false, err
	}
	reqHash := domain.RequestHash(canonBytes)

	// 2. FindByAttempt BEFORE verifying quote/price/stock
	existing, err := s.store.FindByAttempt(ctx, scope, reqID)
	if err != nil {
		return domain.Order{}, false, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}
	if existing != nil {
		if existing.Checkout.RequestHash == reqHash {
			return *existing, false, nil
		}
		return domain.Order{}, false, ErrIdempotencyConflict
	}

	// 3. Verify quote token
	if s.signer == nil {
		return domain.Order{}, false, ErrOrderUnavailable
	}
	now := s.now().UTC()
	quotePayload, err := s.signer.Verify(norm.QuoteToken, now)
	if err != nil {
		if errors.Is(err, domain.ErrQuoteExpired) {
			return domain.Order{}, false, ErrQuoteExpired
		}
		return domain.Order{}, false, domain.NewInvalidFieldError("quote_token", "Mã quote không hợp lệ")
	}

	if err := domain.CompareQuoteChoices(quotePayload, norm.PaymentMethod, norm.Items); err != nil {
		return domain.Order{}, false, ErrQuoteChanged
	}

	// 4. Verify current fee matches quote
	var currentFee domain.Money
	if norm.PaymentMethod == domain.PaymentMethodCashAtCounter {
		currentFee = domain.ZeroMoney
	} else {
		if s.codShippingFee == nil {
			return domain.Order{}, false, ErrShippingUnavailable
		}
		currentFee = *s.codShippingFee
	}
	if quotePayload.ShippingFee != currentFee.String() {
		return domain.Order{}, false, ErrQuoteChanged
	}

	// 5. Run checkout in transaction (with up to 3 attempts for duplicate order numbers)
	productIDs := make([]string, 0, len(norm.Items))
	seenProd := make(map[string]bool)
	for _, it := range norm.Items {
		if !seenProd[it.ProductID] {
			seenProd[it.ProductID] = true
			productIDs = append(productIDs, it.ProductID)
		}
	}

	for attempt := 0; attempt < 3; attempt++ {
		orderNumber := s.newOrderNumber()
		var createdOrder domain.Order

		err = s.store.RunCheckout(ctx, func(ctx context.Context, tx CheckoutTx) error {
			products, err := tx.SellableProducts(ctx, productIDs)
			if err != nil {
				return err
			}

			stockLines := make([]StockLine, len(norm.Items))
			orderItems := make([]domain.OrderItem, len(norm.Items))
			recalculatedSubtotal := domain.ZeroMoney
			prodQtyMap := make(map[string]int64)

			for i, it := range norm.Items {
				prod, ok := products[it.ProductID]
				if !ok {
					return ErrQuoteChanged
				}
				quoteItem := quotePayload.Items[i]
				if prod.Price.String() != quoteItem.UnitPrice {
					return ErrQuoteChanged
				}

				var color, size string
				hasVariants := len(prod.Variants) > 0
				if hasVariants {
					var foundVar *Variant
					for _, v := range prod.Variants {
						if v.SKU == it.SKU {
							foundVar = &v
							break
						}
					}
					if foundVar == nil {
						return ErrQuoteChanged
					}
					if foundVar.Stock < it.Quantity {
						return ErrOutOfStock
					}
					color = foundVar.Color
					size = foundVar.Size
				} else {
					if prod.SKU == "" || prod.SKU != it.SKU {
						return ErrQuoteChanged
					}
					if prod.Stock < it.Quantity {
						return ErrOutOfStock
					}
				}

				prodQtyMap[it.ProductID] += it.Quantity
				stockLines[i] = StockLine{
					ProductID:   it.ProductID,
					SKU:         it.SKU,
					Quantity:    it.Quantity,
					HasVariants: hasVariants,
				}

				lineTotal, err := prod.Price.MulQty(it.Quantity)
				if err != nil {
					return ErrQuoteChanged
				}
				recalculatedSubtotal, err = recalculatedSubtotal.Add(lineTotal)
				if err != nil {
					return ErrQuoteChanged
				}

				orderItems[i] = domain.OrderItem{
					ProductID:   it.ProductID,
					ProductName: prod.Name,
					SKU:         it.SKU,
					Color:       color,
					Size:        size,
					Quantity:    it.Quantity,
					UnitPrice:   prod.Price,
				}
			}

			for prodID, totalQty := range prodQtyMap {
				if products[prodID].Stock < totalQty {
					return ErrOutOfStock
				}
			}

			recalculatedTotal, err := recalculatedSubtotal.Add(currentFee)
			if err != nil || recalculatedTotal.String() != quotePayload.Total {
				return ErrQuoteChanged
			}

			if err := tx.DecrementStock(ctx, stockLines); err != nil {
				return err
			}

			orderNow := s.now().UTC()
			orderToInsert := domain.Order{
				OrderNumber:     orderNumber,
				CustomerID:      principal.CustomerID,
				GuestAccessHash: principal.GuestHash,
				Status:          domain.StatusPending,
				PaymentMethod:   norm.PaymentMethod,
				PaymentStatus:   domain.PaymentUnpaid,
				Subtotal:        recalculatedSubtotal,
				ShippingFee:     currentFee,
				Total:           recalculatedTotal,
				ShippingAddress: norm.ShippingAddress,
				Items:           orderItems,
				Note:            norm.Note,
				Checkout: domain.OrderCheckout{
					Scope:       scope,
					RequestID:   reqID,
					RequestHash: reqHash,
				},
				CreatedAt: orderNow,
				UpdatedAt: orderNow,
			}

			inserted, err := tx.InsertOrder(ctx, orderToInsert)
			if err != nil {
				return err
			}
			createdOrder = inserted
			return nil
		})

		if err == nil {
			return createdOrder, true, nil
		}

		if errors.Is(err, ErrDuplicateOrderNumber) {
			continue
		}

		// 6. Handle duplicate attempt (race condition on same scope/key)
		if errors.Is(err, ErrDuplicateAttempt) {
			existing, findErr := s.store.FindByAttempt(ctx, scope, reqID)
			if findErr == nil && existing != nil {
				if existing.Checkout.RequestHash == reqHash {
					return *existing, false, nil
				}
				return domain.Order{}, false, ErrIdempotencyConflict
			}
			return domain.Order{}, false, ErrOrderUnavailable
		}

		// Business errors return directly
		if errors.Is(err, ErrOutOfStock) || errors.Is(err, ErrQuoteChanged) {
			return domain.Order{}, false, err
		}
		if errors.Is(err, ErrStockInconsistent) {
			return domain.Order{}, false, ErrOrderUnavailable
		}

		// 7. Unknown or commit failure: inspect attempt
		existing, findErr := s.store.FindByAttempt(ctx, scope, reqID)
		if findErr == nil && existing != nil && existing.Checkout.RequestHash == reqHash {
			return *existing, false, nil
		}
		return domain.Order{}, false, ErrOrderUnavailable
	}

	return domain.Order{}, false, ErrOrderUnavailable
}

// Get retrieves an order by ID if viewer is authorized.
func (s *Service) Get(ctx context.Context, viewer Viewer, id string) (*domain.Order, error) {
	order, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}
	if order == nil {
		return nil, ErrNotFound
	}

	if viewer.IsAdmin {
		return order, nil
	}

	if viewer.CustomerID != "" && order.CustomerID != "" && order.CustomerID == viewer.CustomerID {
		return order, nil
	}

	if viewer.GuestHash != "" && order.CustomerID == "" && order.GuestAccessHash != "" {
		if subtle.ConstantTimeCompare([]byte(order.GuestAccessHash), []byte(viewer.GuestHash)) == 1 {
			return order, nil
		}
	}

	return nil, ErrNotFound
}

// List retrieves paginated orders for a customer.
func (s *Service) List(ctx context.Context, customerID string, page, pageSize int64) ([]domain.Order, int64, int64, int64, error) {
	if customerID == "" {
		return nil, 0, 0, 0, ErrNotFound
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	orders, total, err := s.store.ListByCustomer(ctx, customerID, page, pageSize)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}
	return orders, total, page, pageSize, nil
}

// Attempt queries the order outcome of an idempotency attempt for a viewer.
func (s *Service) Attempt(ctx context.Context, viewer Viewer, requestID string) (*domain.Order, error) {
	canonReqID, err := domain.ValidateIdempotencyKey(requestID)
	if err != nil {
		return nil, domain.NewInvalidFieldError("request_id", "Mã yêu cầu (request_id) phải là UUID hợp lệ")
	}

	if viewer.CustomerID != "" {
		scope, err := domain.ValidateCustomerScope(viewer.CustomerID)
		if err == nil {
			order, err := s.store.FindByAttempt(ctx, scope, canonReqID)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
			}
			if order != nil {
				return order, nil
			}
		}
	}
	if viewer.GuestHash != "" {
		scope := "guest:" + viewer.GuestHash
		order, err := s.store.FindByAttempt(ctx, scope, canonReqID)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
		}
		if order != nil {
			return order, nil
		}
	}
	return nil, ErrNotFound
}

// VerifiedPurchase returns true if the viewer is the legitimate owner of the order,
// the order status is "delivered", and the specified product is in the order items.
func (s *Service) VerifiedPurchase(ctx context.Context, viewer Viewer, orderID, productID string) (bool, error) {
	order, err := s.store.FindByID(ctx, orderID)
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrOrderUnavailable, err)
	}
	if order == nil {
		return false, nil
	}

	isOwner := false
	if viewer.CustomerID != "" && order.CustomerID != "" && order.CustomerID == viewer.CustomerID {
		isOwner = true
	} else if viewer.GuestHash != "" && order.CustomerID == "" && order.GuestAccessHash != "" {
		if subtle.ConstantTimeCompare([]byte(order.GuestAccessHash), []byte(viewer.GuestHash)) == 1 {
			isOwner = true
		}
	}

	if !isOwner {
		return false, nil
	}

	if order.Status != domain.StatusDelivered {
		return false, nil
	}

	for _, it := range order.Items {
		if it.ProductID == productID {
			return true, nil
		}
	}
	return false, nil
}
