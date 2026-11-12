package catalog

import (
	"context"
	"errors"
	"sort"
	"strings"

	model "github.com/your-org/aodora/server/internal/domain/catalog"
)

var ErrProductNotFound = errors.New("product not found")

type Repository interface {
	// Categories includes active and inactive documents for ancestor visibility checks.
	Categories(context.Context) ([]model.Category, error)
	ListProducts(context.Context, model.Filter, []string) ([]model.Product, int64, error)
	ProductBySlug(context.Context, string) (*model.Product, error)
	ReviewSummary(context.Context, string) (model.ReviewSummary, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Categories(ctx context.Context) ([]model.Category, error) {
	categories, err := s.repo.Categories(ctx)
	if err != nil {
		return nil, err
	}
	visible := visibleCategories(categories)
	return visible, nil
}

// visibleCategories excludes inactive categories and descendants of inactive,
// deleted or cyclic parents. All public catalog endpoints use the same rule.
func visibleCategories(categories []model.Category) []model.Category {
	byID := make(map[string]model.Category, len(categories))
	for _, c := range categories {
		byID[c.ID] = c
	}
	checked := make(map[string]bool, len(categories))
	active := make(map[string]bool, len(categories))
	visiting := make(map[string]bool, len(categories))
	var visible func(string) bool
	visible = func(id string) bool {
		if checked[id] {
			return active[id]
		}
		c, exists := byID[id]
		if !exists || !c.IsActive || visiting[id] {
			return false
		}
		visiting[id] = true
		ok := c.ParentID == "" || visible(c.ParentID)
		visiting[id] = false
		checked[id] = true
		active[id] = ok
		return ok
	}
	result := make([]model.Category, 0, len(categories))
	for _, c := range categories {
		if visible(c.ID) {
			result = append(result, c)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.ID < b.ID
	})
	return result
}

func (s *Service) List(ctx context.Context, filter model.Filter) (model.ProductPage, error) {
	result := model.ProductPage{
		Data: []model.ProductSummary{}, Page: filter.Page, PageSize: filter.PageSize,
	}
	categories, err := s.Categories(ctx)
	if err != nil {
		return result, err
	}
	ids := make([]string, 0, len(categories))
	requested := strings.TrimSpace(filter.Category)
	for _, c := range categories {
		if requested == "" || strings.EqualFold(requested, c.ID) ||
			strings.EqualFold(requested, c.Key) || strings.EqualFold(requested, c.Slug) {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return result, nil
	}
	products, total, err := s.repo.ListProducts(ctx, filter, ids)
	if err != nil {
		return result, err
	}
	for _, p := range products {
		p.SetAvailability()
		result.Data = append(result.Data, p.Summary())
	}
	result.Total = total
	if total > 0 {
		result.TotalPages = 1 + (total-1)/filter.PageSize
	}
	return result, nil
}

func (s *Service) Detail(ctx context.Context, slug string) (*model.Product, error) {
	product, err := s.repo.ProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	categories, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}
	visible := false
	for _, c := range categories {
		if c.ID == product.CategoryID {
			visible = true
			break
		}
	}
	if !visible {
		return nil, ErrProductNotFound
	}
	summary, err := s.repo.ReviewSummary(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	product.Reviews = summary
	product.SetAvailability()
	return product, nil
}
