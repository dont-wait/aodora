package tailoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

var (
	// ErrCustomOrderExists: the request already has an order that differs from
	// what the caller asked for (an identical retry is replayed instead).
	ErrCustomOrderExists = errors.New("custom order already exists for tailoring request")
	// ErrShippingUnavailable: COD requested but ORDER_COD_SHIPPING_FEE is not configured.
	ErrShippingUnavailable = errors.New("shipping method is unavailable")
	// ErrDuplicateOrderNumber is returned by the store on order_number collision.
	ErrDuplicateOrderNumber = errors.New("duplicate order number")
)

// CustomSKU is the placeholder SKU of a custom line: made-to-measure lines are
// not stock variants and never touch products.stock.
const CustomSKU = "CUSTOM"

const maxOrderNumberAttempts = 3

// CustomOrderInput is the admin-confirmed payload. Price is never read from the product.
type CustomOrderInput struct {
	ProductID     *string
	UnitPrice     string // exact decimal text, validated here
	Quantity      *int64
	PaymentMethod string
	Address       domainorder.ShippingAddressInput // optional; recipient/phone default from the request
	Note          string
}

// CustomOrderDraft is what the store persists. The store fills customer_id and the
// measurement snapshot from the request document it re-reads inside the transaction.
type CustomOrderDraft struct {
	TailoringRequestID string
	ProductID          string
	ProductName        string
	OrderNumber        string
	PaymentMethod      string
	Address            domainorder.NormalizedAddress
	Note               string
	Quantity           int64
	UnitPrice          domainorder.Money
	Subtotal           domainorder.Money
	ShippingFee        domainorder.Money
	Total              domainorder.Money
	Now                time.Time
}

// CustomOrderStore owns the single MongoDB transaction that creates the order.
type CustomOrderStore interface {
	// FindCustomOrder returns the order linked to the request, or (nil, nil).
	FindCustomOrder(ctx context.Context, tailoringRequestID string) (*domainorder.Order, error)
	// CreateCustomOrder atomically (re)checks the request, returns an existing
	// order if any (created=false), otherwise inserts the order and bumps the
	// request's updated_at. It may return ErrNotFound, ErrStateConflict,
	// ErrMeasurementIncomplete or ErrDuplicateOrderNumber.
	CreateCustomOrder(ctx context.Context, d CustomOrderDraft) (order *domainorder.Order, created bool, err error)
}

type ProductLookup interface {
	FindProductByID(ctx context.Context, id string) (*ProductRef, error)
}

type CustomOrderService struct {
	requests interface {
		FindByID(ctx context.Context, id string) (*domain.TailoringRequest, error)
	}
	products       ProductLookup
	store          CustomOrderStore
	codShippingFee *domainorder.Money
	now            func() time.Time
	newOrderNumber func(time.Time) string
}

// NewCustomOrderService: codShippingFee nil means COD is unavailable (Q-04 not
// decided; no default fee). newOrderNumber is the same generator checkout uses
// (order application's DefaultOrderNumber).
func NewCustomOrderService(requests AdminStore, products ProductLookup, store CustomOrderStore, codShippingFee *domainorder.Money, newOrderNumber func(time.Time) string) *CustomOrderService {
	return &CustomOrderService{requests: requests, products: products, store: store, codShippingFee: codShippingFee,
		now: func() time.Time { return time.Now().UTC() }, newOrderNumber: newOrderNumber}
}

func orderInfra(err error) error {
	if err == nil {
		return nil
	}
	for _, b := range []error{ErrNotFound, ErrStateConflict, ErrMeasurementIncomplete, ErrCustomOrderExists, ErrShippingUnavailable} {
		if errors.Is(err, b) {
			return err
		}
	}
	var invalid *domain.InvalidFieldError
	if errors.As(err, &invalid) {
		return err
	}
	return fmt.Errorf("%w: %w", domain.ErrTailoringUnavailable, err)
}

func orderableStatus(s string) bool {
	return s == domain.StatusMeasurementConfirmed || s == domain.StatusInProgress
}

func (s *CustomOrderService) Create(ctx context.Context, id string, in CustomOrderInput) (*domainorder.Order, bool, error) {
	if !validID(id) {
		return nil, false, ErrNotFound
	}
	req, err := s.requests.FindByID(ctx, id)
	if err != nil {
		return nil, false, orderInfra(err)
	}
	if req == nil {
		return nil, false, ErrNotFound
	}

	// Input validation first (400 beats state errors), independent of stored state.
	price, err := domainorder.ParseMoney(strings.TrimSpace(in.UnitPrice))
	if err != nil || price.IsZero() {
		return nil, false, domain.NewInvalidFieldError("unit_price", "Giá phải là số thập phân dương hợp lệ")
	}
	qty := int64(1)
	if in.Quantity != nil {
		qty = *in.Quantity
	}
	if qty < 1 || qty > domainorder.MaxItemQuantity {
		return nil, false, domain.NewInvalidFieldError("quantity", "Số lượng phải từ 1 đến 1.000.000")
	}
	note := strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(note) > 1000 {
		return nil, false, domain.NewInvalidFieldError("note", "Ghi chú không được vượt quá 1000 ký tự")
	}
	var wantProduct string
	if in.ProductID != nil && strings.TrimSpace(*in.ProductID) != "" {
		wantProduct = strings.ToLower(strings.TrimSpace(*in.ProductID))
		if !isValidHexID(wantProduct) {
			return nil, false, domain.NewInvalidFieldError("product_id", "Mã sản phẩm không hợp lệ")
		}
	}
	if req.ProductID != nil {
		if wantProduct != "" && wantProduct != strings.ToLower(*req.ProductID) {
			return nil, false, domain.NewInvalidFieldError("product_id", "Yêu cầu đã gắn với mẫu khác")
		}
		wantProduct = strings.ToLower(*req.ProductID)
	}
	if wantProduct == "" {
		return nil, false, domain.NewInvalidFieldError("product_id", "Yêu cầu chưa có mẫu, cần chọn product_id")
	}
	addr := in.Address
	if strings.TrimSpace(addr.Recipient) == "" {
		addr.Recipient = req.FullName
	}
	if strings.TrimSpace(addr.Phone) == "" {
		addr.Phone = req.Phone
	}
	norm, err := domainorder.Normalize(domainorder.CheckoutInput{
		PaymentMethod:   in.PaymentMethod,
		ShippingAddress: addr,
		Items:           []domainorder.OrderItemInput{{ProductID: wantProduct, SKU: CustomSKU, Quantity: qty}},
		Note:            note,
	})
	if err != nil {
		var inv *domainorder.InvalidFieldError
		if errors.As(err, &inv) {
			return nil, false, domain.NewInvalidFieldError(inv.Field, inv.Message)
		}
		return nil, false, err
	}

	// Replay / conflict detection comes before state checks so a retry after the
	// request moved on (in_progress, completed...) still finds its order.
	existing, err := s.store.FindCustomOrder(ctx, id)
	if err != nil {
		return nil, false, orderInfra(err)
	}
	if existing != nil {
		return s.replay(existing, wantProduct, price, qty, norm.PaymentMethod)
	}

	if !orderableStatus(req.Status) {
		return nil, false, ErrStateConflict
	}
	if !domain.HasAllCMMeasurements(req.Measurements) {
		return nil, false, ErrMeasurementIncomplete
	}
	prod, err := s.products.FindProductByID(ctx, wantProduct)
	if err != nil {
		return nil, false, domain.ErrTailoringUnavailable
	}
	if prod == nil || !prod.IsActive {
		return nil, false, domain.NewInvalidFieldError("product_id", "Sản phẩm không tồn tại")
	}

	fee := domainorder.ZeroMoney
	if norm.PaymentMethod == domainorder.PaymentMethodCOD {
		if s.codShippingFee == nil {
			return nil, false, ErrShippingUnavailable
		}
		fee = *s.codShippingFee
	}
	subtotal, err := price.MulQty(qty)
	if err != nil {
		return nil, false, domain.NewInvalidFieldError("unit_price", "Thành tiền vượt giới hạn")
	}
	total, err := subtotal.Add(fee)
	if err != nil {
		return nil, false, domain.NewInvalidFieldError("unit_price", "Tổng tiền vượt giới hạn")
	}

	for attempt := 0; attempt < maxOrderNumberAttempts; attempt++ {
		now := s.now()
		o, created, err := s.store.CreateCustomOrder(ctx, CustomOrderDraft{
			TailoringRequestID: id, ProductID: prod.ID, ProductName: prod.Name, OrderNumber: s.newOrderNumber(now),
			PaymentMethod: norm.PaymentMethod, Address: norm.ShippingAddress, Note: norm.Note, Quantity: qty,
			UnitPrice: price, Subtotal: subtotal, ShippingFee: fee, Total: total, Now: now,
		})
		if errors.Is(err, ErrDuplicateOrderNumber) {
			continue
		}
		if err != nil {
			return nil, false, orderInfra(err)
		}
		if !created { // lost a race: another request committed first
			return s.replay(o, wantProduct, price, qty, norm.PaymentMethod)
		}
		return o, true, nil
	}
	return nil, false, domain.ErrTailoringUnavailable
}

// replay returns the existing order when the retry asks for the same thing,
// otherwise ErrCustomOrderExists. Only one order per request is ever created
// (also after it was cancelled: re-issuing is a manual decision, see docs).
func (s *CustomOrderService) replay(o *domainorder.Order, productID string, price domainorder.Money, qty int64, method string) (*domainorder.Order, bool, error) {
	same := len(o.Items) == 1 && strings.EqualFold(o.Items[0].ProductID, productID) &&
		o.Items[0].UnitPrice.Equal(price) && o.Items[0].Quantity == qty && o.PaymentMethod == method
	if !same {
		return nil, false, ErrCustomOrderExists
	}
	return o, false, nil
}
