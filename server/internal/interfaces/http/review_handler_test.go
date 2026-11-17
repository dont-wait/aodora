package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestReviewPaginationRejectsInvalidQuery(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/products/sample/reviews?page=abc", nil)
	page, size, field := reviewPagination(request)
	if page != 0 || size != 0 || field != "page" {
		t.Fatalf("got page=%d size=%d field=%q", page, size, field)
	}

	request = httptest.NewRequest("GET", "/api/v1/products/sample/reviews?page_size=51", nil)
	page, size, field = reviewPagination(request)
	if page != 0 || size != 0 || field != "page_size" {
		t.Fatalf("got page=%d size=%d field=%q", page, size, field)
	}
}
