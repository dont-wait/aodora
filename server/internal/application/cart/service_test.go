package cart

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct{}

func (fakeRepository) Cart(context.Context, string) (Cart, error)               { return Cart{}, nil }
func (fakeRepository) AddItem(context.Context, string, Selection) (Cart, error) { return Cart{}, nil }
func (fakeRepository) UpdateItem(context.Context, string, Selection) (Cart, error) {
	return Cart{}, nil
}
func (fakeRepository) RemoveItem(context.Context, string, string, string) (Cart, error) {
	return Cart{}, nil
}
func (fakeRepository) Merge(context.Context, string, string, []Selection) (Cart, error) {
	return Cart{}, nil
}
func (fakeRepository) Validate(context.Context, []Selection) (Cart, error) { return Cart{}, nil }

func TestRejectsInvalidCartSelection(t *testing.T) {
	service := NewService(fakeRepository{})
	_, err := service.AddItem(context.Background(), "customer", Selection{ProductID: "product", SKU: "SKU", Quantity: 0, FulfillmentMode: "ready"})
	var invalid Invalid
	if !errors.As(err, &invalid) || invalid.Field != "quantity" {
		t.Fatalf("error = %v", err)
	}
}

func TestAcceptsReadySelection(t *testing.T) {
	service := NewService(fakeRepository{})
	_, err := service.AddItem(context.Background(), "customer", Selection{ProductID: "product", SKU: "SKU", Quantity: 1, FulfillmentMode: "ready"})
	if err != nil {
		t.Fatal(err)
	}
}
