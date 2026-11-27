package tailoring

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFullName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		field   string
	}{
		{"valid", "Nguyễn Văn A", "Nguyễn Văn A", false, ""},
		{"valid trimmed", "  Trần Thị B  ", "Trần Thị B", false, ""},
		{"empty", "", "", true, "full_name"},
		{"whitespace only", "   \t\n  ", "", true, "full_name"},
		{"too long", strings.Repeat("A", 201), "", true, "full_name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateFullName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateFullName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var ifErr *InvalidFieldError
				if !errors.As(err, &ifErr) || ifErr.Field != tt.field {
					t.Fatalf("expected InvalidFieldError with field %q, got %v", tt.field, err)
				}
			} else if got != tt.want {
				t.Fatalf("ValidateFullName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid standard", "0901234567", "0901234567", false},
		{"valid with spaces", "090 123 4567", "090 123 4567", false},
		{"valid with plus", "+84901234567", "+84901234567", false},
		{"valid trimmed", "  0901234567  ", "0901234567", false},
		{"empty", "", "", true},
		{"whitespace only", "   ", "", true},
		{"letters", "090abc1234", "", true},
		{"too short", "123", "", true},
		{"too long", strings.Repeat("1", 51), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePhone(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePhone() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Fatalf("ValidatePhone() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidatePreferredContact(t *testing.T) {
	pPhone := "phone"
	pZalo := "zalo"
	pEmail := "email"
	pWhitespace := "   "

	tests := []struct {
		name    string
		input   *string
		wantErr bool
		field   string
	}{
		{"nil", nil, false, ""},
		{"phone", &pPhone, false, ""},
		{"zalo", &pZalo, false, ""},
		{"whitespace (treated as nil)", &pWhitespace, false, ""},
		{"email invalid", &pEmail, true, "preferred_contact"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePreferredContact(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePreferredContact() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var ifErr *InvalidFieldError
				if !errors.As(err, &ifErr) || ifErr.Field != tt.field {
					t.Fatalf("expected field %q, got %v", tt.field, err)
				}
			} else if tt.input != nil && *tt.input != pWhitespace && (got == nil || *got != *tt.input) {
				t.Fatalf("ValidatePreferredContact() = %v, want %v", got, tt.input)
			}
		})
	}
}

func TestValidateMeasurements(t *testing.T) {
	fValid := 34.5
	fZero := 0.0
	fNegative := -5.0
	fTooLarge := 600.0
	bodyText := "1m65 / 50kg"
	bodyTooLong := strings.Repeat("M", 501)

	tests := []struct {
		name     string
		input    *Measurements
		wantErr  bool
		errField string
		wantBody bool
		wantNeck bool
	}{
		{"nil input", nil, false, "", false, false},
		{"empty measurements", &Measurements{}, false, "", false, false},
		{"valid body only", &Measurements{Body: &bodyText}, false, "", true, false},
		{"valid measurements exact", &Measurements{NeckCM: &fValid, Body: &bodyText}, false, "", true, true},
		{"zero measurement invalid", &Measurements{NeckCM: &fZero}, true, "neck_cm", false, false},
		{"negative measurement invalid", &Measurements{BustCM: &fNegative}, true, "bust_cm", false, false},
		{"too large measurement invalid", &Measurements{GarmentLengthCM: &fTooLarge}, true, "garment_length_cm", false, false},
		{"body too long", &Measurements{Body: &bodyTooLong}, true, "body", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateMeasurements(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateMeasurements() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				var ifErr *InvalidFieldError
				if !errors.As(err, &ifErr) || ifErr.Field != tt.errField {
					t.Fatalf("expected error on field %q, got %v", tt.errField, err)
				}
			} else if got != nil {
				if tt.wantBody && (got.Body == nil || *got.Body != bodyText) {
					t.Errorf("got.Body = %v, want %q", got.Body, bodyText)
				}
				if tt.wantNeck && (got.NeckCM == nil || *got.NeckCM != fValid) {
					t.Errorf("got.NeckCM = %v, want %f", got.NeckCM, fValid)
				}
			}
		})
	}
}
