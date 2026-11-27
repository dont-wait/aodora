package catalog

import "strings"

// StandardSizes is the only list of sizes the owner may assign to a variant.
// The order is the display order.
var StandardSizes = []string{"XS", "S", "M", "L", "XL", "XXL", "Freesize"}

// CanonicalSize returns the standard spelling of size (case-insensitive).
// The empty string is valid and means "product without a size".
func CanonicalSize(size string) (string, bool) {
	size = strings.TrimSpace(size)
	if size == "" {
		return "", true
	}
	for _, standard := range StandardSizes {
		if strings.EqualFold(size, standard) {
			return standard, true
		}
	}
	return "", false
}

// SizeMessage is the validation message shown for an unsupported size.
const SizeMessage = "Cỡ phải là một trong XS, S, M, L, XL, XXL, Freesize"
