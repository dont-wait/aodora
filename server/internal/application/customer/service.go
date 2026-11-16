package customer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound        = errors.New("customer data not found")
	ErrProductNotFound = errors.New("wishlist product not found")
	ErrConflict        = errors.New("customer data changed")
)

type Invalid struct {
	Field   string
	Message string
}

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

type Address struct {
	ID        string `json:"id" bson:"id,omitempty"`
	Recipient string `json:"recipient" bson:"recipient"`
	Phone     string `json:"phone" bson:"phone"`
	Line1     string `json:"line1" bson:"line1"`
	Ward      string `json:"ward,omitempty" bson:"ward,omitempty"`
	District  string `json:"district,omitempty" bson:"district,omitempty"`
	City      string `json:"city" bson:"city"`
	IsDefault bool   `json:"is_default" bson:"is_default"`
}

type Measurements struct {
	Body            *string  `json:"body,omitempty" bson:"body,omitempty"`
	NeckCM          *float64 `json:"neck_cm,omitempty" bson:"neck_cm,omitempty"`
	BustCM          *float64 `json:"bust_cm,omitempty" bson:"bust_cm,omitempty"`
	WaistCM         *float64 `json:"waist_cm,omitempty" bson:"waist_cm,omitempty"`
	HipsCM          *float64 `json:"hips_cm,omitempty" bson:"hips_cm,omitempty"`
	ShoulderCM      *float64 `json:"shoulder_cm,omitempty" bson:"shoulder_cm,omitempty"`
	SleeveCM        *float64 `json:"sleeve_cm,omitempty" bson:"sleeve_cm,omitempty"`
	GarmentLengthCM *float64 `json:"garment_length_cm,omitempty" bson:"garment_length_cm,omitempty"`
}

type Profile struct {
	ID               string        `json:"id"`
	FullName         string        `json:"full_name"`
	Email            string        `json:"email"`
	Phone            string        `json:"phone,omitempty"`
	Addresses        []Address     `json:"addresses"`
	Measurements     *Measurements `json:"measurement_profile,omitempty"`
	MeasurementNotes string        `json:"measurement_notes,omitempty"`
	Version          int64         `json:"-"`
}

type ProfileInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

type AddressInput struct {
	Recipient string `json:"recipient"`
	Phone     string `json:"phone"`
	Line1     string `json:"line1"`
	Ward      string `json:"ward"`
	District  string `json:"district"`
	City      string `json:"city"`
	IsDefault bool   `json:"is_default"`
}

type WishlistProduct struct {
	ID             string `json:"id"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Material       string `json:"material,omitempty"`
	Price          int64  `json:"price"`
	Image          *Image `json:"image,omitempty"`
	Stock          int64  `json:"stock"`
	ReadyAvailable bool   `json:"ready_available"`
	Bespoke        bool   `json:"bespoke"`
	Unavailable    bool   `json:"unavailable"`
}

type Image struct {
	Src string `json:"src" bson:"src"`
	Alt string `json:"alt" bson:"alt"`
}

type WishlistItem struct {
	ID      string           `json:"id"`
	Product *WishlistProduct `json:"product,omitempty"`
}

type Repository interface {
	Profile(context.Context, string) (Profile, error)
	UpdateProfile(context.Context, string, ProfileInput) (Profile, error)
	ReplaceAddresses(context.Context, string, []Address, int64) (Profile, error)
	ReplaceMeasurements(context.Context, string, *Measurements, string, int64) (Profile, error)
	Wishlist(context.Context, string) ([]WishlistItem, error)
	AddWishlist(context.Context, string, string) error
	RemoveWishlist(context.Context, string, string) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Profile(ctx context.Context, userID string) (Profile, error) {
	return s.repo.Profile(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, input ProfileInput) (Profile, error) {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Phone = strings.TrimSpace(input.Phone)
	if utf8.RuneCountInString(input.FullName) < 1 || utf8.RuneCountInString(input.FullName) > 150 {
		return Profile{}, Invalid{Field: "full_name", Message: "Họ tên phải từ 1 đến 150 ký tự"}
	}
	if utf8.RuneCountInString(input.Phone) > 30 {
		return Profile{}, Invalid{Field: "phone", Message: "Số điện thoại quá dài"}
	}
	return s.repo.UpdateProfile(ctx, userID, input)
}

func (s *Service) AddAddress(ctx context.Context, userID string, input AddressInput) (Profile, error) {
	if err := validateAddress(input); err != nil {
		return Profile{}, err
	}
	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	addresses := normalizeAddresses(profile.Addresses)
	input = cleanAddressInput(input)
	address := Address{ID: newAddressID(), Recipient: input.Recipient, Phone: input.Phone, Line1: input.Line1, Ward: input.Ward, District: input.District, City: input.City, IsDefault: input.IsDefault}
	if len(addresses) == 0 {
		address.IsDefault = true
	}
	if address.IsDefault {
		for i := range addresses {
			addresses[i].IsDefault = false
		}
	}
	addresses = append(addresses, address)
	return s.repo.ReplaceAddresses(ctx, userID, addresses, profile.Version)
}

func (s *Service) UpdateAddress(ctx context.Context, userID, addressID string, input AddressInput) (Profile, error) {
	if err := validateAddress(input); err != nil {
		return Profile{}, err
	}
	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	addresses := normalizeAddresses(profile.Addresses)
	index := addressIndex(addresses, addressID)
	if index < 0 {
		return Profile{}, ErrNotFound
	}
	input = cleanAddressInput(input)
	addresses[index] = Address{ID: addresses[index].ID, Recipient: input.Recipient, Phone: input.Phone, Line1: input.Line1, Ward: input.Ward, District: input.District, City: input.City, IsDefault: input.IsDefault}
	if input.IsDefault {
		for i := range addresses {
			addresses[i].IsDefault = i == index
		}
	}
	return s.repo.ReplaceAddresses(ctx, userID, normalizeAddresses(addresses), profile.Version)
}

func (s *Service) DeleteAddress(ctx context.Context, userID, addressID string) (Profile, error) {
	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	addresses := normalizeAddresses(profile.Addresses)
	index := addressIndex(addresses, addressID)
	if index < 0 {
		return Profile{}, ErrNotFound
	}
	addresses = append(addresses[:index], addresses[index+1:]...)
	return s.repo.ReplaceAddresses(ctx, userID, normalizeAddresses(addresses), profile.Version)
}

func (s *Service) Measurements(ctx context.Context, userID string) (Profile, error) {
	return s.repo.Profile(ctx, userID)
}

func (s *Service) UpdateMeasurements(ctx context.Context, userID string, measurements *Measurements, notes string) (Profile, error) {
	if err := validateMeasurements(measurements, notes); err != nil {
		return Profile{}, err
	}
	profile, err := s.repo.Profile(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	return s.repo.ReplaceMeasurements(ctx, userID, measurements, strings.TrimSpace(notes), profile.Version)
}

func (s *Service) Wishlist(ctx context.Context, userID string) ([]WishlistItem, error) {
	return s.repo.Wishlist(ctx, userID)
}

func (s *Service) AddWishlist(ctx context.Context, userID, productID string) error {
	return s.repo.AddWishlist(ctx, userID, productID)
}

func (s *Service) RemoveWishlist(ctx context.Context, userID, productID string) error {
	return s.repo.RemoveWishlist(ctx, userID, productID)
}

func validateAddress(input AddressInput) error {
	for field, value := range map[string]string{"recipient": input.Recipient, "phone": input.Phone, "line1": input.Line1, "city": input.City} {
		if strings.TrimSpace(value) == "" {
			return Invalid{Field: field, Message: "Trường này là bắt buộc"}
		}
		if utf8.RuneCountInString(strings.TrimSpace(value)) > 200 {
			return Invalid{Field: field, Message: "Giá trị quá dài"}
		}
	}
	if utf8.RuneCountInString(input.Ward) > 100 || utf8.RuneCountInString(input.District) > 100 {
		return Invalid{Field: "address", Message: "Thông tin địa chỉ quá dài"}
	}
	return nil
}

func validateMeasurements(measurements *Measurements, notes string) error {
	if measurements == nil {
		if utf8.RuneCountInString(notes) > 2000 {
			return Invalid{Field: "measurement_notes", Message: "Ghi chú quá dài"}
		}
		return nil
	}
	values := map[string]*float64{"neck_cm": measurements.NeckCM, "bust_cm": measurements.BustCM, "waist_cm": measurements.WaistCM, "hips_cm": measurements.HipsCM, "shoulder_cm": measurements.ShoulderCM, "sleeve_cm": measurements.SleeveCM, "garment_length_cm": measurements.GarmentLengthCM}
	for field, value := range values {
		if value != nil && (*value < 0 || *value > 1000) {
			return Invalid{Field: field, Message: "Số đo phải từ 0 đến 1000 cm"}
		}
	}
	if measurements.Body != nil && utf8.RuneCountInString(*measurements.Body) > 500 {
		return Invalid{Field: "body", Message: "Mô tả vóc dáng quá dài"}
	}
	if utf8.RuneCountInString(notes) > 2000 {
		return Invalid{Field: "measurement_notes", Message: "Ghi chú quá dài"}
	}
	return nil
}

func cleanAddressInput(input AddressInput) AddressInput {
	input.Recipient, input.Phone, input.Line1 = strings.TrimSpace(input.Recipient), strings.TrimSpace(input.Phone), strings.TrimSpace(input.Line1)
	input.Ward, input.District, input.City = strings.TrimSpace(input.Ward), strings.TrimSpace(input.District), strings.TrimSpace(input.City)
	return input
}

func normalizeAddresses(addresses []Address) []Address {
	result := append([]Address(nil), addresses...)
	hasDefault := false
	for i := range result {
		if result[i].ID == "" {
			result[i].ID = fmt.Sprintf("legacy-%d", i)
		}
		if result[i].IsDefault && !hasDefault {
			hasDefault = true
		} else if result[i].IsDefault {
			result[i].IsDefault = false
		}
	}
	if len(result) > 0 && !hasDefault {
		result[0].IsDefault = true
	}
	return result
}

func addressIndex(addresses []Address, id string) int {
	for i := range addresses {
		if addresses[i].ID == id {
			return i
		}
	}
	return -1
}

func newAddressID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("address-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(value)
}
