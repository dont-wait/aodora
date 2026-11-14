package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	model "github.com/your-org/aodora/server/internal/domain/catalog"
)

type repositoryStub struct {
	categories []model.Category
	products   []model.Product
	summary    model.ReviewSummary
	total      int64
	ids        []string
	fail       error
}

func (r *repositoryStub) Categories(context.Context) ([]model.Category, error) {
	if r.fail != nil {
		return nil, r.fail
	}
	return r.categories, nil
}
func (r *repositoryStub) ListProducts(_ context.Context, _ model.Filter, ids []string) ([]model.Product, int64, error) {
	r.ids = ids
	if r.fail != nil {
		return nil, 0, r.fail
	}
	return r.products, r.total, nil
}
func (r *repositoryStub) ProductBySlug(_ context.Context, slug string) (*model.Product, error) {
	if r.fail != nil {
		return nil, r.fail
	}
	for _, product := range r.products {
		if product.Slug == slug {
			copied := product
			return &copied, nil
		}
	}
	return nil, nil
}
func (r *repositoryStub) ReviewSummary(context.Context, string) (model.ReviewSummary, error) {
	return r.summary, r.fail
}

func categoriesForTest() []model.Category {
	return []model.Category{
		{ID: "root", Name: "Cổ Phục", Slug: "co-phuc", Key: "co-phuc", IsActive: true, SortOrder: 2},
		{ID: "child", Name: "Con", ParentID: "root", IsActive: true, SortOrder: 3},
		{ID: "inactive", Name: "Ẩn", IsActive: false},
		{ID: "hidden-child", Name: "Con ẩn", ParentID: "inactive", IsActive: true},
		{ID: "orphan", ParentID: "does-not-exist", IsActive: true},
		{ID: "cycle-a", ParentID: "cycle-b", IsActive: true},
		{ID: "cycle-b", ParentID: "cycle-a", IsActive: true},
	}
}

func TestVisibleCategoriesFiltersAncestorsCyclesAndSorts(t *testing.T) {
	got := visibleCategories(categoriesForTest())
	if len(got) != 2 || got[0].ID != "root" || got[1].ID != "child" {
		t.Fatalf("visible category IDs = %+v, want root and child", got)
	}
}

func TestListRestrictsCategoryAndPages(t *testing.T) {
	repo := &repositoryStub{
		categories: categoriesForTest(), total: 11,
		products: []model.Product{{ID: "p1", CategoryID: "root", Slug: "ao-dai", Name: "Áo dài",
			Price: json.Number("2150000"), Stock: 2,
			Variants: []model.Variant{{SKU: "A-M", Color: "Trắng", Size: "M", Stock: 0},
				{SKU: "A-L", Color: "Trắng", Size: "L", Stock: 2}},
		}},
	}
	result, err := NewService(repo).List(context.Background(), model.Filter{
		Category: "co-phuc", Page: 2, PageSize: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repo.ids, []string{"root"}) {
		t.Fatalf("category filter %v", repo.ids)
	}
	if result.Total != 11 || result.Page != 2 || result.TotalPages != 3 {
		t.Fatalf("page: %+v", result)
	}
	if !result.Data[0].ReadyAvailable {
		t.Fatal("expected purchasable product")
	}
	// Name/price are values from the repository, never from a fixed card list.
	if result.Data[0].Price.String() != "2150000" {
		t.Fatal("wrong price")
	}
}

func TestListUnknownOrHiddenCategoryEmpty(t *testing.T) {
	repo := &repositoryStub{categories: categoriesForTest()}
	for _, category := range []string{"unknown", "inactive", "hidden-child"} {
		t.Run(category, func(t *testing.T) {
			result, err := NewService(repo).List(context.Background(), model.Filter{
				Category: category, Page: 1, PageSize: 12,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Total != 0 || len(result.Data) != 0 {
				t.Fatalf("unexpected products: %+v", result)
			}
		})
	}
}

func TestDetailRefusesProductInInactiveCategory(t *testing.T) {
	repo := &repositoryStub{
		categories: categoriesForTest(),
		products:   []model.Product{{ID: "p1", Slug: "hidden-product", CategoryID: "hidden-child"}},
	}
	_, err := NewService(repo).Detail(context.Background(), "hidden-product")
	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("got %v, want not found", err)
	}
}

func TestDetailHasOwnStoryVariantsReviewSummary(t *testing.T) {
	repo := &repositoryStub{
		categories: categoriesForTest(), summary: model.ReviewSummary{Count: 2, Average: 4.5},
		products: []model.Product{
			{ID: "a", Slug: "ao-a", CategoryID: "root", Story: &model.Story{Title: "Truyện mẫu A"}},
			{ID: "b", Slug: "ao-b", CategoryID: "root", Stock: 4,
				Variants: []model.Variant{{SKU: "B-M", Size: "M", Color: "Ngà", Stock: 0},
					{SKU: "B-L", Size: "L", Color: "Ngà", Stock: 4}},
				Story: &model.Story{Title: "Truyện mẫu B"}},
		},
	}
	p, err := NewService(repo).Detail(context.Background(), "ao-b")
	if err != nil {
		t.Fatal(err)
	}
	if p.Story.Title != "Truyện mẫu B" || p.Reviews.Count != 2 {
		t.Fatalf("bad detail: %+v", p)
	}
	if p.Variants[0].Available || !p.Variants[1].Available {
		t.Fatal("stock/SKU availability wrong")
	}
}

func TestRepositoryFailurePropagates(t *testing.T) {
	repo := &repositoryStub{fail: errors.New("db failed")}
	if _, err := NewService(repo).Categories(context.Background()); err == nil {
		t.Fatal("expected db error")
	}
}
