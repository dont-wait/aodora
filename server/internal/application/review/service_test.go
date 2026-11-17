package review

import (
	"context"
	"errors"
	"testing"

	apporder "github.com/your-org/aodora/server/internal/application/order"
)

type fakeRepository struct{}

func (fakeRepository) List(context.Context, string, int64, int64) (Page, error) { return Page{}, nil }
func (fakeRepository) Own(context.Context, string, string) (*OwnReview, error) {
	return nil, ErrNotFound
}
func (fakeRepository) Upsert(context.Context, string, string, Input) (OwnReview, error) {
	return OwnReview{}, nil
}

type fakeVerifier struct{ allowed bool }

func (v fakeVerifier) VerifiedPurchase(context.Context, apporder.Viewer, string, string) (bool, error) {
	return v.allowed, nil
}

func TestSubmitRequiresDeliveredPurchase(t *testing.T) {
	service := NewService(fakeRepository{}, fakeVerifier{})
	_, err := service.Submit(context.Background(), "customer", "product", Input{OrderID: "order", Rating: 5})
	if !errors.Is(err, ErrNotEligible) {
		t.Fatalf("error = %v, want not eligible", err)
	}
}

func TestRejectsInvalidRatingAndTooManyImages(t *testing.T) {
	service := NewService(fakeRepository{}, fakeVerifier{allowed: true})
	_, err := service.Submit(context.Background(), "customer", "product", Input{OrderID: "order", Rating: 6})
	var invalid Invalid
	if !errors.As(err, &invalid) || invalid.Field != "rating" {
		t.Fatalf("rating error = %v", err)
	}
	images := make([]Image, 6)
	_, err = service.Submit(context.Background(), "customer", "product", Input{OrderID: "order", Rating: 5, Images: images})
	if !errors.As(err, &invalid) || invalid.Field != "images" {
		t.Fatalf("image error = %v", err)
	}
}
