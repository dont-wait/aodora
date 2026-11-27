package catalog

import "testing"

func TestCanonicalSize(t *testing.T) {
	for in, want := range map[string]string{"xs": "XS", " m ": "M", "FREESIZE": "Freesize", "xxl": "XXL", "": "", "  ": ""} {
		if got, ok := CanonicalSize(in); !ok || got != want {
			t.Errorf("CanonicalSize(%q) = %q,%v want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"A", "AA", "XXXL", "Size M"} {
		if _, ok := CanonicalSize(in); ok {
			t.Errorf("CanonicalSize(%q) accepted", in)
		}
	}
}
