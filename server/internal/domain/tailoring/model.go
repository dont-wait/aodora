package tailoring

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Allowed tailoring request statuses in schema
const (
	StatusSubmitted             = "submitted"
	StatusContacted             = "contacted"
	StatusConsultationScheduled = "consultation_scheduled"
	StatusMeasurementConfirmed  = "measurement_confirmed"
	StatusInProgress            = "in_progress"
	StatusCompleted             = "completed"
	StatusCancelled             = "cancelled"
)

// Allowed preferred contact channels
const (
	ContactPhone = "phone"
	ContactZalo  = "zalo"
)

// Validation limits
const (
	MaxFullNameLen   = 200
	MaxPhoneLen      = 50
	MaxBodyLen       = 500
	MaxNotesLen      = 2000
	MaxMeasurementCM = 500.0
)

var phonePattern = regexp.MustCompile(`^[0-9+() \-\.]{8,20}$`)

// ErrTailoringUnavailable signals a transient infrastructure failure when
// looking up a product. The handler maps this to 503 TAILORING_UNAVAILABLE.
var ErrTailoringUnavailable = errors.New("tailoring service temporarily unavailable")

// Measurements captures the 8 tailoring measurement fields according to schema.
// All measurement _cm fields must be positive numbers and are optional.
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

// TailoringRequest represents a customer's custom tailoring request.
type TailoringRequest struct {
	ID               string        `json:"id" bson:"_id,omitempty"`
	CustomerID       *string       `json:"customer_id,omitempty" bson:"customer_id,omitempty"`
	FullName         string        `json:"full_name" bson:"full_name"`
	Phone            string        `json:"phone" bson:"phone"`
	PreferredContact *string       `json:"preferred_contact,omitempty" bson:"preferred_contact,omitempty"`
	CategoryID       *string       `json:"category_id,omitempty" bson:"category_id,omitempty"`
	ProductID        *string       `json:"product_id,omitempty" bson:"product_id,omitempty"`
	ProductSlug      *string       `json:"product_slug,omitempty" bson:"product_slug,omitempty"`
	Measurements     *Measurements `json:"measurements,omitempty" bson:"measurements,omitempty"`
	Notes            *string       `json:"notes,omitempty" bson:"notes,omitempty"`
	Status           string        `json:"status" bson:"status"`
	AppointmentAt    *time.Time    `json:"appointment_at,omitempty" bson:"appointment_at,omitempty"`
	AssignedStaff    *string       `json:"assigned_staff,omitempty" bson:"assigned_staff,omitempty"`
	CreatedAt        time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at" bson:"updated_at"`
}

// InvalidFieldError represents a field-level validation error matching API conventions.
type InvalidFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *InvalidFieldError) Error() string {
	return e.Field + ": " + e.Message
}

func NewInvalidFieldError(field, message string) *InvalidFieldError {
	return &InvalidFieldError{
		Field:   field,
		Message: message,
	}
}

// ValidateFullName validates and normalizes the full name.
func ValidateFullName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewInvalidFieldError("full_name", "Họ và tên là bắt buộc")
	}
	if utf8.RuneCountInString(trimmed) > MaxFullNameLen {
		return "", NewInvalidFieldError("full_name", "Họ và tên không được vượt quá 200 ký tự")
	}
	return trimmed, nil
}

// ValidatePhone validates and normalizes the phone number.
func ValidatePhone(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", NewInvalidFieldError("phone", "Số điện thoại là bắt buộc")
	}
	if utf8.RuneCountInString(trimmed) > MaxPhoneLen || !phonePattern.MatchString(trimmed) {
		return "", NewInvalidFieldError("phone", "Số điện thoại không hợp lệ")
	}
	return trimmed, nil
}

// ValidatePreferredContact validates the preferred contact method if provided.
func ValidatePreferredContact(contact *string) (*string, error) {
	if contact == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*contact)
	if trimmed == "" {
		return nil, nil
	}
	switch trimmed {
	case ContactPhone, ContactZalo:
		return &trimmed, nil
	default:
		return nil, NewInvalidFieldError("preferred_contact", "Kênh liên hệ phải là phone hoặc zalo")
	}
}

// ValidateNotes validates optional notes.
func ValidateNotes(notes *string) (*string, error) {
	if notes == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*notes)
	if trimmed == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(trimmed) > MaxNotesLen {
		return nil, NewInvalidFieldError("notes", "Ghi chú không được vượt quá 2000 ký tự")
	}
	return &trimmed, nil
}

// ValidateMeasurements validates body string and 7 numeric measurements.
// Missing measurements remain nil (never converted to 0).
// Provided numbers must be positive (> 0) and within a reasonable limit (<= 500 cm).
func ValidateMeasurements(m *Measurements) (*Measurements, error) {
	if m == nil {
		return nil, nil
	}

	var out Measurements
	hasData := false

	if m.Body != nil {
		trimmedBody := strings.TrimSpace(*m.Body)
		if trimmedBody != "" {
			if utf8.RuneCountInString(trimmedBody) > MaxBodyLen {
				return nil, NewInvalidFieldError("body", "Mô tả vóc dáng không được vượt quá 500 ký tự")
			}
			out.Body = &trimmedBody
			hasData = true
		}
	}

	fields := []struct {
		name  string
		value *float64
		dest  **float64
	}{
		{"neck_cm", m.NeckCM, &out.NeckCM},
		{"bust_cm", m.BustCM, &out.BustCM},
		{"waist_cm", m.WaistCM, &out.WaistCM},
		{"hips_cm", m.HipsCM, &out.HipsCM},
		{"shoulder_cm", m.ShoulderCM, &out.ShoulderCM},
		{"sleeve_cm", m.SleeveCM, &out.SleeveCM},
		{"garment_length_cm", m.GarmentLengthCM, &out.GarmentLengthCM},
	}

	for _, f := range fields {
		if f.value != nil {
			v := *f.value
			if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 || v > MaxMeasurementCM {
				return nil, NewInvalidFieldError(f.name, "Số đo phải là số dương hợp lệ (lớn hơn 0 và tối đa 500 cm)")
			}
			val := v
			*f.dest = &val
			hasData = true
		}
	}

	if !hasData {
		return nil, nil
	}

	return &out, nil
}
