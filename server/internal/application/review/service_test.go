package review

import (
	"context"
	"errors"
	"testing"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
)

type fakeRepository struct {
	upserts      *[]Input
	guestUpserts *[]string
	guestInputs  *[]GuestInput
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
func (fakeRepository) GuestReviews(context.Context, string) ([]OwnReview, error) { return nil, nil }
func (r fakeRepository) GuestUpsert(_ context.Context, orderID, productID, name string, input GuestInput) (OwnReview, error) {
	if r.guestUpserts != nil {
		*r.guestUpserts = append(*r.guestUpserts, orderID+"|"+productID+"|"+name)
		*r.guestInputs = append(*r.guestInputs, input)
	}
	return OwnReview{}, nil
}
func (fakeRepository) AdminList(_ context.Context, f AdminFilter) (AdminPage, error) {
	return AdminPage{Page: f.Page, Size: f.Size}, nil
}
func (fakeRepository) Delete(context.Context, string) error { return nil }

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
		return &domainorder.Order{ID: id, Status: status, GuestAccessHash: "guest-a", Items: items, ShippingAddress: domainorder.NormalizedAddress{Recipient: " Nguyễn An "}}
	case id == "account-order" && viewer.CustomerID == "cust-a":
		return &domainorder.Order{ID: id, Status: status, CustomerID: "cust-a", Items: items}
	}
	return nil
}

func (o fakeOrders) Get(_ context.Context, viewer apporder.Viewer, id string) (*domainorder.Order, error) {
	if order := o.order(viewer, id); order != nil {
		return order, nil
	}
	return nil, apporder.ErrNotFound
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
		_, err = service.SubmitGuest(context.Background(), "guest-a", "guest-order", "p1", GuestInput{Rating: rating})
		if !errors.As(err, &invalid) || invalid.Field != "rating" {
			t.Fatalf("guest rating %d error = %v", rating, err)
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

func TestSubmitGuestUsesOrderTokenAndRecipientName(t *testing.T) {
	var keys []string
	var inputs []GuestInput
	service := NewService(fakeRepository{guestUpserts: &keys, guestInputs: &inputs}, fakeOrders{})
	if _, err := service.SubmitGuest(context.Background(), "guest-a", "guest-order", "p1", GuestInput{Rating: 5}); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != "guest-order|p1|Nguyễn An" || inputs[0].Images == nil {
		t.Fatalf("guest upsert = %v %+v", keys, inputs)
	}
}

func TestSubmitGuestRejectsWrongTokenUndeliveredAndForeignProduct(t *testing.T) {
	var keys []string
	var inputs []GuestInput
	repo := fakeRepository{guestUpserts: &keys, guestInputs: &inputs}
	ctx := context.Background()
	if _, err := NewService(repo, fakeOrders{}).SubmitGuest(ctx, "guest-b", "guest-order", "p1", GuestInput{Rating: 5}); !errors.Is(err, ErrOrderAccess) {
		t.Fatalf("wrong token error = %v", err)
	}
	if _, err := NewService(repo, fakeOrders{}).SubmitGuest(ctx, "", "guest-order", "p1", GuestInput{Rating: 5}); !errors.Is(err, ErrOrderAccess) {
		t.Fatalf("missing token error = %v", err)
	}
	// An account order cannot be reviewed through the guest route.
	if _, err := NewService(repo, fakeOrders{}).SubmitGuest(ctx, "guest-a", "account-order", "p1", GuestInput{Rating: 5}); !errors.Is(err, ErrOrderAccess) {
		t.Fatalf("account order via guest error = %v", err)
	}
	if _, err := NewService(repo, fakeOrders{status: domainorder.StatusPending}).SubmitGuest(ctx, "guest-a", "guest-order", "p1", GuestInput{Rating: 5}); !errors.Is(err, ErrNotEligible) {
		t.Fatalf("undelivered error = %v", err)
	}
	if _, err := NewService(repo, fakeOrders{}).SubmitGuest(ctx, "guest-a", "guest-order", "p2", GuestInput{Rating: 5}); !errors.Is(err, ErrNotEligible) {
		t.Fatalf("foreign product error = %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("rejected guest submissions wrote %v", keys)
	}
}

func TestGuestReviewsRequireOrderAccess(t *testing.T) {
	service := NewService(fakeRepository{}, fakeOrders{})
	if _, err := service.GuestReviews(context.Background(), "guest-b", "guest-order"); !errors.Is(err, ErrOrderAccess) {
		t.Fatalf("error = %v", err)
	}
	if _, err := service.GuestReviews(context.Background(), "guest-a", "guest-order"); err != nil {
		t.Fatal(err)
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
