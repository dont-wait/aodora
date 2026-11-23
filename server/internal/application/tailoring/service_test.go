package tailoring

import (
	"context"
	"errors"
	"testing"

	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
)

type mockRepository struct {
	created   *domaintailoring.TailoringRequest
	products  map[string]*ProductRef
	slugs     map[string]*ProductRef
	findErr   error
	createErr error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		products: make(map[string]*ProductRef),
		slugs:    make(map[string]*ProductRef),
	}
}

func (m *mockRepository) Create(_ context.Context, req *domaintailoring.TailoringRequest) (*domaintailoring.TailoringRequest, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	clone := *req
	clone.ID = "6d0000000000000000000099"
	m.created = &clone
	return &clone, nil
}

func (m *mockRepository) FindProductByID(_ context.Context, id string) (*ProductRef, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.products[id], nil
}

func (m *mockRepository) FindProductBySlug(_ context.Context, slug string) (*ProductRef, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.slugs[slug], nil
}

func TestSubmit_GuestMinimal(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	res, err := svc.Submit(context.Background(), SubmitInput{
		FullName: "Nguyễn Văn A",
		Phone:    "0900000001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "6d0000000000000000000099" {
		t.Errorf("expected ID %q, got %q", "6d0000000000000000000099", res.ID)
	}
	if res.Status != domaintailoring.StatusSubmitted {
		t.Errorf("expected status %q, got %q", domaintailoring.StatusSubmitted, res.Status)
	}
	if res.CustomerID != nil {
		t.Errorf("expected nil CustomerID for guest, got %v", *res.CustomerID)
	}
	if res.Measurements != nil {
		t.Errorf("expected nil measurements, got %v", res.Measurements)
	}
}

func TestSubmit_AuthenticatedCustomer(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	custID := "690000000000000000000001"
	res, err := svc.Submit(context.Background(), SubmitInput{
		CustomerID: &custID,
		FullName:   "Trần Thị B",
		Phone:      "0900000002",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CustomerID == nil || *res.CustomerID != custID {
		t.Errorf("expected CustomerID %q, got %v", custID, res.CustomerID)
	}
}

func TestSubmit_MissingNameOrPhone(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.Submit(context.Background(), SubmitInput{
		FullName: "   ",
		Phone:    "0900000001",
	})
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	var ifErr *domaintailoring.InvalidFieldError
	if !errors.As(err, &ifErr) || ifErr.Field != "full_name" {
		t.Fatalf("expected error on full_name, got %v", err)
	}

	_, err = svc.Submit(context.Background(), SubmitInput{
		FullName: "Nguyễn Văn A",
		Phone:    "   ",
	})
	if err == nil {
		t.Fatal("expected error for empty phone, got nil")
	}
	if !errors.As(err, &ifErr) || ifErr.Field != "phone" {
		t.Fatalf("expected error on phone, got %v", err)
	}
}

func TestSubmit_ProductVerification(t *testing.T) {
	repo := newMockRepository()
	prodID := "680000000000000000000001"
	prodSlug := "ao-dai-lua-ha-dong"
	repo.products[prodID] = &ProductRef{
		ID:         prodID,
		Slug:       prodSlug,
		CategoryID: "co-phuc",
		Name:       "Áo dài lụa Hà Đông",
		IsActive:   true,
	}
	repo.slugs[prodSlug] = repo.products[prodID]

	svc := NewService(repo)

	// Valid product ID + matching slug
	res, err := svc.Submit(context.Background(), SubmitInput{
		FullName:    "Lê Văn C",
		Phone:       "0900000003",
		ProductID:   &prodID,
		ProductSlug: &prodSlug,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ProductID == nil || *res.ProductID != prodID {
		t.Errorf("expected ProductID %q, got %v", prodID, res.ProductID)
	}
	if res.ProductSlug == nil || *res.ProductSlug != prodSlug {
		t.Errorf("expected ProductSlug %q, got %v", prodSlug, res.ProductSlug)
	}
	if res.CategoryID == nil || *res.CategoryID != "co-phuc" {
		t.Errorf("expected CategoryID %q, got %v", "co-phuc", res.CategoryID)
	}

	// Product ID and slug mismatch
	wrongSlug := "ao-dai-cach-tan"
	_, err = svc.Submit(context.Background(), SubmitInput{
		FullName:    "Lê Văn C",
		Phone:       "0900000003",
		ProductID:   &prodID,
		ProductSlug: &wrongSlug,
	})
	if err == nil {
		t.Fatal("expected error on slug mismatch, got nil")
	}
	var ifErr *domaintailoring.InvalidFieldError
	if !errors.As(err, &ifErr) || ifErr.Field != "product_slug" {
		t.Fatalf("expected error on product_slug, got %v", err)
	}

	// Non-existent product ID
	nonExistentID := "680000000000000000000099"
	_, err = svc.Submit(context.Background(), SubmitInput{
		FullName:  "Lê Văn C",
		Phone:     "0900000003",
		ProductID: &nonExistentID,
	})
	if err == nil {
		t.Fatal("expected error on non-existent product ID, got nil")
	}
	if !errors.As(err, &ifErr) || ifErr.Field != "product_id" {
		t.Fatalf("expected error on product_id, got %v", err)
	}

	// Invalid hex product ID
	invalidHex := "not-a-valid-hex-id!"
	_, err = svc.Submit(context.Background(), SubmitInput{
		FullName:  "Lê Văn C",
		Phone:     "0900000003",
		ProductID: &invalidHex,
	})
	if err == nil {
		t.Fatal("expected error on invalid hex product ID, got nil")
	}
	if !errors.As(err, &ifErr) || ifErr.Field != "product_id" {
		t.Fatalf("expected error on product_id, got %v", err)
	}

	// Product slug only
	res, err = svc.Submit(context.Background(), SubmitInput{
		FullName:    "Lê Văn C",
		Phone:       "0900000003",
		ProductSlug: &prodSlug,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ProductID == nil || *res.ProductID != prodID {
		t.Errorf("expected ProductID filled from slug, got %v", res.ProductID)
	}

	// Non-existent product slug only
	nonExistentSlug := "non-existent-slug"
	_, err = svc.Submit(context.Background(), SubmitInput{
		FullName:    "Lê Văn C",
		Phone:       "0900000003",
		ProductSlug: &nonExistentSlug,
	})
	if err == nil {
		t.Fatal("expected error on non-existent product slug, got nil")
	}
	if !errors.As(err, &ifErr) || ifErr.Field != "product_slug" {
		t.Fatalf("expected error on product_slug, got %v", err)
	}
}

func TestSubmit_ProductInfraErrorIsUnavailable(t *testing.T) {
	prodID := "680000000000000000000001"
	slug := "ao-dai"
	for name, in := range map[string]SubmitInput{
		"by id":   {FullName: "A", Phone: "0900000001", ProductID: &prodID},
		"by slug": {FullName: "A", Phone: "0900000001", ProductSlug: &slug},
	} {
		repo := newMockRepository()
		repo.findErr = errors.New("connection refused")
		_, err := NewService(repo).Submit(context.Background(), in)
		if !errors.Is(err, domaintailoring.ErrTailoringUnavailable) {
			t.Fatalf("%s: expected ErrTailoringUnavailable, got %v", name, err)
		}
		var ifErr *domaintailoring.InvalidFieldError
		if errors.As(err, &ifErr) {
			t.Fatalf("%s: infra error must not be InvalidFieldError", name)
		}
		if repo.created != nil {
			t.Fatalf("%s: must not create", name)
		}
	}
}

func TestSubmit_CreateErrorIsUnavailable(t *testing.T) {
	repo := newMockRepository()
	repo.createErr = errors.New("write failed")
	_, err := NewService(repo).Submit(context.Background(), SubmitInput{FullName: "A", Phone: "0900000001"})
	if !errors.Is(err, domaintailoring.ErrTailoringUnavailable) {
		t.Fatalf("expected ErrTailoringUnavailable, got %v", err)
	}
}

func TestSubmit_InactiveProductRejected(t *testing.T) {
	repo := newMockRepository()
	prodID := "680000000000000000000002"
	slug := "ao-an"
	p := &ProductRef{ID: prodID, Slug: slug, IsActive: false}
	repo.products[prodID] = p
	repo.slugs[slug] = p
	svc := NewService(repo)

	var ifErr *domaintailoring.InvalidFieldError
	_, err := svc.Submit(context.Background(), SubmitInput{FullName: "A", Phone: "0900000001", ProductID: &prodID})
	if !errors.As(err, &ifErr) || ifErr.Field != "product_id" || ifErr.Message != "Sản phẩm không tồn tại" {
		t.Fatalf("expected product_id not-exist error, got %v", err)
	}
	_, err = svc.Submit(context.Background(), SubmitInput{FullName: "A", Phone: "0900000001", ProductSlug: &slug})
	if !errors.As(err, &ifErr) || ifErr.Field != "product_slug" {
		t.Fatalf("expected product_slug error, got %v", err)
	}
	if repo.created != nil {
		t.Fatal("must not create")
	}
}
