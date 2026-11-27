package order

import (
	"context"

	domain "github.com/your-org/aodora/server/internal/domain/order"
)

type Variant struct {
	SKU   string
	Color string
	Size  string
	Stock int64
}

// Product is a snapshot of an actively sellable public product.
type Product struct {
	ID       string
	Name     string
	SKU      string // Product-level SKU (for legacy products without variants)
	Price    domain.Money
	Stock    int64
	Variants []Variant
}

type StockLine struct {
	ProductID   string
	SKU         string
	Quantity    int64
	HasVariants bool
}

// Catalog reads sellable products outside of transaction, used for quotation.
type Catalog interface {
	SellableProducts(ctx context.Context, productIDs []string) (map[string]Product, error)
}

// CheckoutTx executes all operations within the SAME transaction.
type CheckoutTx interface {
	SellableProducts(ctx context.Context, productIDs []string) (map[string]Product, error)
	DecrementStock(ctx context.Context, lines []StockLine) error
	InsertOrder(ctx context.Context, o domain.Order) (domain.Order, error)
}

// Store handles checkout transaction execution and order retrieval.
type Store interface {
	RunCheckout(ctx context.Context, fn func(ctx context.Context, tx CheckoutTx) error) error
	FindByAttempt(ctx context.Context, scope, requestID string) (*domain.Order, error)
	FindByID(ctx context.Context, id string) (*domain.Order, error)
	ListByCustomer(ctx context.Context, customerID, status string, page, pageSize int64) ([]domain.Order, int64, error)
}
