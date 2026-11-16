package customer

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	profile Profile
}

func (f *fakeRepository) Profile(context.Context, string) (Profile, error) {
	if f.profile.ID == "" {
		return Profile{}, ErrNotFound
	}
	return f.profile, nil
}
func (f *fakeRepository) UpdateProfile(_ context.Context, _ string, input ProfileInput) (Profile, error) {
	f.profile.FullName, f.profile.Phone = input.FullName, input.Phone
	return f.profile, nil
}
func (f *fakeRepository) ReplaceAddresses(_ context.Context, _ string, addresses []Address, _ int64) (Profile, error) {
	f.profile.Addresses = addresses
	return f.profile, nil
}
func (f *fakeRepository) ReplaceMeasurements(_ context.Context, _ string, measurements *Measurements, notes string, _ int64) (Profile, error) {
	f.profile.Measurements, f.profile.MeasurementNotes = measurements, notes
	return f.profile, nil
}
func (f *fakeRepository) Wishlist(context.Context, string) ([]WishlistItem, error) { return nil, nil }
func (f *fakeRepository) AddWishlist(context.Context, string, string) error        { return nil }
func (f *fakeRepository) RemoveWishlist(context.Context, string, string) error     { return nil }

func TestAddressOperationsKeepOneDefault(t *testing.T) {
	repo := &fakeRepository{profile: Profile{ID: "customer-1"}}
	service := NewService(repo)
	first, err := service.AddAddress(context.Background(), "customer-1", AddressInput{Recipient: "A", Phone: "0901", Line1: "1 Main", City: "Hà Nội"})
	if err != nil || len(first.Addresses) != 1 || !first.Addresses[0].IsDefault {
		t.Fatalf("first address = %+v, err = %v", first, err)
	}
	second, err := service.AddAddress(context.Background(), "customer-1", AddressInput{Recipient: "B", Phone: "0902", Line1: "2 Main", City: "Hà Nội", IsDefault: true})
	if err != nil || second.Addresses[0].IsDefault || !second.Addresses[1].IsDefault {
		t.Fatalf("default address was not replaced: %+v, err = %v", second, err)
	}
	updated, err := service.UpdateAddress(context.Background(), "customer-1", second.Addresses[0].ID, AddressInput{Recipient: "A2", Phone: "0903", Line1: "3 Main", City: "Hà Nội", IsDefault: true})
	if err != nil || !updated.Addresses[0].IsDefault || updated.Addresses[1].IsDefault {
		t.Fatalf("updated default address = %+v, err = %v", updated, err)
	}
	deleted, err := service.DeleteAddress(context.Background(), "customer-1", updated.Addresses[0].ID)
	if err != nil || len(deleted.Addresses) != 1 || !deleted.Addresses[0].IsDefault {
		t.Fatalf("deleted address result = %+v, err = %v", deleted, err)
	}
}

func TestMeasurementRejectsNegativeValue(t *testing.T) {
	negative := -1.0
	service := NewService(&fakeRepository{profile: Profile{ID: "customer-1"}})
	_, err := service.UpdateMeasurements(context.Background(), "customer-1", &Measurements{BustCM: &negative}, "")
	var invalid Invalid
	if !errors.As(err, &invalid) || invalid.Field != "bust_cm" {
		t.Fatalf("error = %v", err)
	}
}

func TestProfileUpdateDoesNotAcceptEmptyName(t *testing.T) {
	service := NewService(&fakeRepository{profile: Profile{ID: "customer-1"}})
	_, err := service.UpdateProfile(context.Background(), "customer-1", ProfileInput{FullName: "  "})
	var invalid Invalid
	if !errors.As(err, &invalid) || invalid.Field != "full_name" {
		t.Fatalf("error = %v", err)
	}
}
