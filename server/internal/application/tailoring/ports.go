package tailoring

import (
	"context"

	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
)

// ProductRef holds minimal product reference data for validation.
type ProductRef struct {
	ID         string
	Slug       string
	CategoryID string
	Name       string
	IsActive   bool
}

// Repository defines data access operations required for tailoring requests.
type Repository interface {
	Create(ctx context.Context, req *domaintailoring.TailoringRequest) (*domaintailoring.TailoringRequest, error)
	FindProductByID(ctx context.Context, id string) (*ProductRef, error)
	FindProductBySlug(ctx context.Context, slug string) (*ProductRef, error)
}
