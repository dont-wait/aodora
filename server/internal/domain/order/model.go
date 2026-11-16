package order

import "time"

const (
	StatusPending   = "pending"
	PaymentUnpaid   = "unpaid"
	StatusDelivered = "delivered"
)

type OrderItem struct {
	ProductID   string
	ProductName string
	SKU         string
	Color       string
	Size        string
	Quantity    int64
	UnitPrice   Money
}

type OrderCheckout struct {
	Scope       string
	RequestID   string
	RequestHash string
}

type Order struct {
	ID              string
	OrderNumber     string
	CustomerID      string // empty for guest orders
	GuestAccessHash string
	Status          string
	PaymentMethod   string
	PaymentStatus   string
	Subtotal        Money
	ShippingFee     Money
	Total           Money
	ShippingAddress NormalizedAddress
	Items           []OrderItem
	Note            string
	Checkout        OrderCheckout
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
