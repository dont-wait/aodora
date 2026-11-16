package order

import (
	"errors"
)

var (
	ErrMoneyOverflow     = errors.New("money overflow: exceeds 34 significant digits")
	ErrInvalidMoney      = errors.New("invalid money format")
	ErrInvalidGuestToken = errors.New("invalid guest access token")
	ErrInvalidScope      = errors.New("invalid scope")
	ErrQuoteInvalid      = errors.New("quote token is invalid")
	ErrQuoteExpired      = errors.New("quote token has expired")
	ErrQuoteChanged      = errors.New("quote choices have changed")
)

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
