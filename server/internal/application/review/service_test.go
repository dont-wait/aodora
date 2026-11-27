package review

import (
	"context"
	"errors"
	"fmt"
	"testing"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

type fakeRepository struct {
	upserts *[]Input
}

func (fakeRepository) List(context.Context, string, int64, int64) (Page, error) { return Page{}, nil }
func (fakeRepository) Own(context.Context, string, string) (*OwnReview, error) {
	return nil, ErrNotFound
}
func (r fakeRepository) Upsert(_ context.Context, _, _ string, input Input) (OwnReview, error) {
	if r.upserts != nil {
		*r.upserts = append(*r.upserts, input)
	}
	return OwnReview{}, nil
}
func (fakeRepository) Statuses(_ context.Context, _ string, ids []string) ([]ProductReviewStatus, error) {
	out := make([]ProductReviewStatus, len(ids))
	for i, id := range ids {
		out[i] = ProductReviewStatus{ProductID: id}
	}
	return out, nil
}
func (fakeRepository) AdminList(_ context.Context, f AdminFilter) (AdminPage, error) {
	return AdminPage{Page: f.Page, Size: f.Size}, nil
}
func (fakeRepository) Delete(context.Context, string) error          { return nil }
func (fakeRepository) SetHidden(context.Context, string, bool) error { return nil }

// fakeOrders models one guest order owned by hash "guest-a" and one account
// order owned by customer "cust-a"; both contain product "p1".
type fakeOrders struct {
	status string
}

func (o fakeOrders) order(viewer apporder.Viewer, id string) *domainorder.Order {
	status := o.status
	if status == "" {
		status = domainorder.StatusDelivered
	}
	items := []domainorder.OrderItem{{ProductID: "p1"}}
	switch {
	case id == "guest-order" && viewer.GuestHash == "guest-a":
		return &domainorder.Order{ID: id, Status: status, GuestAccessHash: "guest-a", Items: items}
	case id == "account-order" && viewer.CustomerID == "cust-a":
		return &domainorder.Order{ID: id, Status: status, CustomerID: "cust-a", Items: items}
	}
	return nil
}

func (o fakeOrders) VerifiedPurchase(_ context.Context, viewer apporder.Viewer, id, productID string) (bool, error) {
	order := o.order(viewer, id)
	if order == nil || order.Status != domainorder.StatusDelivered {
		return false, nil
	}
	for _, item := range order.Items {
		if item.ProductID == productID {
			return true, nil
		}
	}
	return false, nil
}

func TestSubmitRequiresDeliveredOwnedOrderWithProduct(t *testing.T) {
	for name, tc := range map[string]struct {
		customer, order, product, status string
	}{
		"not delivered":     {"cust-a", "account-order", "p1", domainorder.StatusShipping},
		"cancelled":         {"cust-a", "account-order", "p1", domainorder.StatusCancelled},
		"product not in it": {"cust-a", "account-order", "p2", ""},
		"other customer":    {"cust-b", "account-order", "p1", ""},
		"guest order":       {"cust-a", "guest-order", "p1", ""},
	} {
		t.Run(name, func(t *testing.T) {
			var upserts []Input
			service := NewService(fakeRepository{upserts: &upserts}, fakeOrders{status: tc.status})
			_, err := service.Submit(context.Background(), tc.customer, tc.product, Input{OrderID: tc.order, Rating: 5})
			if !errors.Is(err, ErrNotEligible) || len(upserts) != 0 {
				t.Fatalf("err=%v upserts=%d, want not eligible and no write", err, len(upserts))
			}
		})
	}
}

func TestSubmitNormalizesImagesToEmptyArray(t *testing.T) {
	var upserts []Input
	service := NewService(fakeRepository{upserts: &upserts}, fakeOrders{})
	if _, err := service.Submit(context.Background(), "cust-a", "p1", Input{OrderID: "account-order", Rating: 4, Content: "  tốt  "}); err != nil {
		t.Fatal(err)
	}
	if len(upserts) != 1 || upserts[0].Images == nil || upserts[0].Content != "tốt" {
		t.Fatalf("upsert = %+v, want trimmed content and non-nil images", upserts)
	}
}

func TestRejectsInvalidRatingAndImages(t *testing.T) {
	service := NewService(fakeRepository{}, fakeOrders{})
	for _, rating := range []int64{0, 6, -1} {
		_, err := service.Submit(context.Background(), "cust-a", "p1", Input{OrderID: "account-order", Rating: rating})
		var invalid Invalid
		if !errors.As(err, &invalid) || invalid.Field != "rating" {
			t.Fatalf("rating %d error = %v", rating, err)
		}
	}
	for _, images := range [][]Image{
		make([]Image, 6),
		{{Src: "javascript:alert(1)", Alt: "x"}},
		{{Src: "https://cdn.example/a.jpg", Alt: ""}},
	} {
		_, err := service.Submit(context.Background(), "cust-a", "p1", Input{OrderID: "account-order", Rating: 5, Images: images})
		var invalid Invalid
		if !errors.As(err, &invalid) || invalid.Field != "images" {
			t.Fatalf("images %v error = %v", images, err)
		}
	}
}

func TestAdminListValidatesAndClampsFilter(t *testing.T) {
	service := NewService(fakeRepository{}, fakeOrders{})
	var invalid Invalid
	if _, err := service.AdminList(context.Background(), AdminFilter{Rating: 6}); !errors.As(err, &invalid) {
		t.Fatalf("rating filter error = %v", err)
	}
	page, err := service.AdminList(context.Background(), AdminFilter{Page: 0, Size: 500})
	if err != nil || page.Page != 1 || page.Size != 100 {
		t.Fatalf("page = %+v err = %v", page, err)
	}
}

func TestStatusesDedupesAndBoundsInput(t *testing.T) {
	service := NewService(fakeRepository{}, fakeOrders{})
	got, err := service.Statuses(context.Background(), "cust-a", []string{"p1", " p1 ", "", "p2"})
	if err != nil || len(got) != 2 || got[0].ProductID != "p1" || got[1].ProductID != "p2" {
		t.Fatalf("statuses = %+v err = %v", got, err)
	}
	if got, err := service.Statuses(context.Background(), "cust-a", nil); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("empty statuses = %#v err = %v", got, err)
	}
	many := make([]string, MaxStatusProducts+1)
	for i := range many {
		many[i] = fmt.Sprintf("p%d", i)
	}
	var invalid Invalid
	if _, err := service.Statuses(context.Background(), "cust-a", many); !errors.As(err, &invalid) {
		t.Fatalf("too many ids error = %v", err)
	}
}
