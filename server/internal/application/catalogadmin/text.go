package catalogadmin

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// foldText lower-cases and strips Vietnamese diacritics ("Đỏ Đô" -> "do do").
func foldText(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		switch r {
		case 'đ', 'Đ':
			r = 'd'
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// normalizeSKU upper-cases and strips diacritics and spaces: "ao dai-đỏ" -> "AO-DAI-DO".
func normalizeSKU(s string) string {
	folded := foldText(s)
	folded = strings.ReplaceAll(folded, " ", "-")
	return strings.ToUpper(folded)
}
