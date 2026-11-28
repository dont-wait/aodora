package order

import "time"

const (
	StatusPending   = "pending"
	PaymentUnpaid   = "unpaid"
	StatusDelivered = "delivered"
)

// MeasurementProfile is an immutable snapshot of tailoring measurements copied
// into a custom order. Absent values stay nil (never 0).
type MeasurementProfile struct {
	Body            *string
	NeckCM          *float64
	BustCM          *float64
	WaistCM         *float64
	HipsCM          *float64
	ShoulderCM      *float64
	SleeveCM        *float64
	GarmentLengthCM *float64
}

type OrderItem struct {
	Measurements    *MeasurementProfile
	FulfillmentMode string
	ProductID       string
	ProductName     string
	SKU             string
	Color           string
	Size            string
	Quantity        int64
	UnitPrice       Money
}

type OrderCheckout struct {
	Scope       string
	RequestID   string
	RequestHash string
}

type Order struct {
	FulfillmentMode      string // "ready", "custom" or empty for legacy
	TailoringRequestID   string // set only for custom orders
	MeasurementProfile   *MeasurementProfile
	CancelledAt          *time.Time
	InventoryRestock     string
	InventoryRestockedAt *time.Time
	PaidAt               *time.Time
	RefundedAt           *time.Time
	DeliveredAt          *time.Time // set when the shop marks the order delivered; nil on older orders
	ID                   string
	OrderNumber          string
	CustomerID           string // empty for guest orders
	GuestAccessHash      string
	Status               string
	PaymentMethod        string
	PaymentStatus        string
	Subtotal             Money
	ShippingFee          Money
	Total                Money
	ShippingAddress      NormalizedAddress
	Items                []OrderItem
	Note                 string
	Checkout             OrderCheckout
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
