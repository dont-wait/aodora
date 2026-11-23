package tailoring

import (
	"context"
	"strings"
	"time"

	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
)

// Service implements tailoring application use cases.
type Service struct {
	repo Repository
}

// NewService creates a new tailoring application service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SubmitInput represents input parameters for creating a new tailoring request.
type SubmitInput struct {
	CustomerID       *string
	FullName         string
	Phone            string
	PreferredContact *string
	CategoryID       *string
	ProductID        *string
	ProductSlug      *string
	Measurements     *domaintailoring.Measurements
	Notes            *string
}

// Submit validates and creates a new tailoring request.
func (s *Service) Submit(ctx context.Context, input SubmitInput) (*domaintailoring.TailoringRequest, error) {
	fullName, err := domaintailoring.ValidateFullName(input.FullName)
	if err != nil {
		return nil, err
	}

	phone, err := domaintailoring.ValidatePhone(input.Phone)
	if err != nil {
		return nil, err
	}

	preferredContact, err := domaintailoring.ValidatePreferredContact(input.PreferredContact)
	if err != nil {
		return nil, err
	}

	notes, err := domaintailoring.ValidateNotes(input.Notes)
	if err != nil {
		return nil, err
	}

	measurements, err := domaintailoring.ValidateMeasurements(input.Measurements)
	if err != nil {
		return nil, err
	}

	var productID *string
	var productSlug *string
	var categoryID *string

	if input.CategoryID != nil {
		trimmedCat := strings.TrimSpace(*input.CategoryID)
		if trimmedCat != "" {
			categoryID = &trimmedCat
		}
	}

	if input.ProductID != nil && strings.TrimSpace(*input.ProductID) != "" {
		rawID := strings.TrimSpace(*input.ProductID)
		if !isValidHexID(rawID) {
			return nil, domaintailoring.NewInvalidFieldError("product_id", "Mã sản phẩm không hợp lệ")
		}

		prod, err := s.repo.FindProductByID(ctx, rawID)
		if err != nil {
			return nil, domaintailoring.ErrTailoringUnavailable
		}
		if prod == nil || !prod.IsActive {
			return nil, domaintailoring.NewInvalidFieldError("product_id", "Sản phẩm không tồn tại")
		}

		if input.ProductSlug != nil && strings.TrimSpace(*input.ProductSlug) != "" {
			rawSlug := strings.TrimSpace(*input.ProductSlug)
			if rawSlug != prod.Slug {
				return nil, domaintailoring.NewInvalidFieldError("product_slug", "Mã sản phẩm và đường dẫn không khớp")
			}
		}

		productID = &prod.ID
		productSlug = &prod.Slug
		if categoryID == nil && prod.CategoryID != "" {
			categoryID = &prod.CategoryID
		}
	} else if input.ProductSlug != nil && strings.TrimSpace(*input.ProductSlug) != "" {
		rawSlug := strings.TrimSpace(*input.ProductSlug)
		prod, err := s.repo.FindProductBySlug(ctx, rawSlug)
		if err != nil {
			return nil, domaintailoring.ErrTailoringUnavailable
		}
		if prod == nil || !prod.IsActive {
			return nil, domaintailoring.NewInvalidFieldError("product_slug", "Sản phẩm không tồn tại")
		}

		productID = &prod.ID
		productSlug = &prod.Slug
		if categoryID == nil && prod.CategoryID != "" {
			categoryID = &prod.CategoryID
		}
	}

	var customerID *string
	if input.CustomerID != nil {
		trimmedCust := strings.TrimSpace(*input.CustomerID)
		if trimmedCust != "" {
			customerID = &trimmedCust
		}
	}

	now := time.Now().UTC()
	req := &domaintailoring.TailoringRequest{
		CustomerID:       customerID,
		FullName:         fullName,
		Phone:            phone,
		PreferredContact: preferredContact,
		CategoryID:       categoryID,
		ProductID:        productID,
		ProductSlug:      productSlug,
		Measurements:     measurements,
		Notes:            notes,
		Status:           domaintailoring.StatusSubmitted,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	created, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, domaintailoring.ErrTailoringUnavailable
	}
	return created, nil
}

func isValidHexID(s string) bool {
	if len(s) != 24 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
