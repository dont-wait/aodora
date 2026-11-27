package catalogadmin

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// memRepo is a small in-memory Repository so import rules can be tested end to end.
type memRepo struct {
	fakeRepo
	cats     []map[string]any
	products []map[string]any
	next     int
	log      []string
}

func (m *memRepo) id() string {
	m.next++
	return fmt.Sprintf("%024x", m.next)
}
func (m *memRepo) Categories(context.Context) ([]map[string]any, error) { return m.cats, nil }
func (m *memRepo) CreateCategory(_ context.Context, v CategoryInput) (map[string]any, error) {
	c := map[string]any{"id": m.id(), "name": v.Name, "slug": v.Slug, "description": v.Description, "parent_id": v.ParentID, "is_active": v.IsActive, "sort_order": v.SortOrder, "version": int64(0)}
	if v.Image != nil {
		c["image"] = map[string]any{"src": v.Image.Src, "alt": v.Image.Alt}
	}
	m.cats = append(m.cats, c)
	m.log = append(m.log, "create-category "+v.Slug)
	return c, nil
}
func (m *memRepo) UpdateCategory(_ context.Context, id string, v CategoryInput) (map[string]any, error) {
	for _, c := range m.cats {
		if c["id"] == id {
			if *v.ExpectedVersion != c["version"].(int64) {
				return nil, ErrConflict
			}
			c["name"], c["description"], c["parent_id"], c["is_active"], c["sort_order"] = v.Name, v.Description, v.ParentID, v.IsActive, v.SortOrder
			c["version"] = c["version"].(int64) + 1
			m.log = append(m.log, "update-category "+v.Slug)
			return c, nil
		}
	}
	return nil, ErrNotFound
}
func (m *memRepo) ProductState(_ context.Context, id string) (ProductState, error) {
	p := m.find(id)
	if p == nil {
		return ProductState{}, ErrNotFound
	}
	active, _ := p["is_active"].(bool)
	state := ProductState{ID: id, Slug: p["slug"].(string), Active: active}
	if cs, ok := p["colors"].([]any); ok {
		for _, c := range cs {
			state.Colors = append(state.Colors, c.(map[string]any)["name"].(string))
		}
	}
	for _, v := range p["variants"].([]any) {
		mv := v.(map[string]any)
		state.Variants = append(state.Variants, StateVariant{SKU: mv["sku"].(string), Color: mv["color"].(string), Size: mv["size"].(string), Stock: mv["stock"].(int64)})
	}
	return state, nil
}
func (m *memRepo) ProductsBySlug(_ context.Context, slugs []string) ([]map[string]any, error) {
	var out []map[string]any
	for _, p := range m.products {
		for _, s := range slugs {
			if p["slug"] == s {
				out = append(out, p)
			}
		}
	}
	return out, nil
}
func (m *memRepo) SKUOwners(_ context.Context, skus []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range m.products {
		for _, v := range p["variants"].([]any) {
			for _, s := range skus {
				if strings.EqualFold(v.(map[string]any)["sku"].(string), s) {
					out[strings.ToLower(s)] = p["slug"].(string)
				}
			}
		}
	}
	return out, nil
}
func (m *memRepo) CreateProduct(_ context.Context, v ProductCreate) (map[string]any, error) {
	variants := []any{}
	for _, x := range v.Variants {
		variants = append(variants, map[string]any{"sku": x.SKU, "color": x.Color, "size": x.Size, "stock": x.Stock})
	}
	var colors []any
	for _, c := range v.Colors {
		colors = append(colors, map[string]any{"name": c.Name, "hex": c.Hex})
	}
	p := map[string]any{"id": m.id(), "slug": v.Slug, "name": v.Name, "category_id": v.CategoryID, "description": v.Description, "material": v.Material,
		"price": v.Price, "is_active": v.IsActive, "variants": variants, "colors": colors, "version": int64(0), "images": []any{}}
	if v.CompareAtPrice != nil {
		p["compare_at_price"] = *v.CompareAtPrice
	}
	m.products = append(m.products, p)
	m.log = append(m.log, "create-product "+v.Slug)
	return p, nil
}
func (m *memRepo) find(id string) map[string]any {
	for _, p := range m.products {
		if p["id"] == id {
			return p
		}
	}
	return nil
}
func (m *memRepo) UpdateProduct(_ context.Context, id string, v ProductInput) (map[string]any, error) {
	p := m.find(id)
	if p == nil {
		return nil, ErrNotFound
	}
	if *v.ExpectedVersion != p["version"].(int64) {
		return nil, ErrConflict
	}
	p["name"], p["description"], p["material"], p["price"], p["is_active"], p["category_id"] = v.Name, v.Description, v.Material, v.Price, v.IsActive, v.CategoryID
	colors := []any{}
	for _, c := range v.Colors {
		colors = append(colors, map[string]any{"name": c.Name, "hex": c.Hex})
	}
	p["colors"] = colors
	p["version"] = p["version"].(int64) + 1
	m.log = append(m.log, "update-product "+v.Slug)
	return p, nil
}
func (m *memRepo) AddVariant(_ context.Context, id string, v Variant) (map[string]any, error) {
	p := m.find(id)
	p["variants"] = append(p["variants"].([]any), map[string]any{"sku": v.SKU, "color": v.Color, "size": v.Size, "stock": v.Stock})
	m.log = append(m.log, "add-variant "+v.SKU)
	return p, nil
}
func (m *memRepo) UpdateVariant(_ context.Context, id, sku string, v VariantDetails) (map[string]any, error) {
	p := m.find(id)
	for _, x := range p["variants"].([]any) {
		if mv := x.(map[string]any); mv["sku"] == sku {
			mv["color"], mv["size"] = v.Color, v.Size
		}
	}
	m.log = append(m.log, "update-variant "+sku)
	return p, nil
}
func (m *memRepo) AdjustInventory(_ context.Context, id, sku string, delta int64, _ string) (map[string]any, error) {
	p := m.find(id)
	for _, x := range p["variants"].([]any) {
		if mv := x.(map[string]any); mv["sku"] == sku {
			mv["stock"] = mv["stock"].(int64) + delta
		}
	}
	m.log = append(m.log, fmt.Sprintf("adjust %s %+d", sku, delta))
	return p, nil
}
func (m *memRepo) Products(_ context.Context, page, size int64) (ProductPage, error) {
	return ProductPage{Items: m.products, Total: int64(len(m.products)), Page: page, PageSize: size, TotalPages: 1}, nil
}

func parseCSV(t *testing.T, text string, allowed, required []string) []Row {
	t.Helper()
	rows, errs := ParseRows([]byte(text), "csv", allowed, required)
	if len(errs) > 0 {
		t.Fatalf("parse: %+v", errs)
	}
	return rows
}

const catHeader = "name,slug,description,parent_slug,sort_order,is_active,image_url,image_alt\n"
const prodHeader = "product_slug,name,category_slug,description,material,price,compare_at_price,is_active,image_url,gallery_urls,color_name,color_hex,size,sku,stock\n"

func TestParseRowsHandlesBOMAndMissingOrUnknownColumns(t *testing.T) {
	rows, errs := ParseRows([]byte("\xef\xbb\xbf"+catHeader+"Áo,ao,,,1,true,,\n,,,,,,,\n"), "csv", categoryColumns, []string{"name", "slug"})
	if len(errs) != 0 || len(rows) != 1 || rows[0].Number != 2 || rows[0].get("name") != "Áo" {
		t.Fatalf("rows=%+v errs=%+v", rows, errs)
	}
	_, errs = ParseRows([]byte("name,description\nA,B\n"), "csv", categoryColumns, []string{"name", "slug"})
	if len(errs) != 1 || errs[0].Field != "slug" || errs[0].Message != "Thiếu cột bắt buộc" {
		t.Fatalf("missing column: %+v", errs)
	}
	_, errs = ParseRows([]byte("name,slug,colour\nA,a,x\n"), "csv", categoryColumns, []string{"name", "slug"})
	if len(errs) != 1 || errs[0].Field != "colour" {
		t.Fatalf("unknown column: %+v", errs)
	}
	if _, errs = ParseRows([]byte(""), "csv", categoryColumns, nil); len(errs) == 0 {
		t.Fatal("empty file must fail")
	}
	if _, errs = ParseRows([]byte(catHeader), "csv", categoryColumns, nil); len(errs) == 0 {
		t.Fatal("header-only file must fail")
	}
	big := catHeader + strings.Repeat("A,a,,,,,,\n", MaxImportRows+1)
	if _, errs = ParseRows([]byte(big), "csv", categoryColumns, nil); len(errs) == 0 || !strings.Contains(errs[0].Message, "Tối đa") {
		t.Fatalf("row limit: %+v", errs)
	}
}

func TestParseRowsJSON(t *testing.T) {
	rows, errs := ParseRows([]byte(`[{"name":"Áo","slug":"ao","sort_order":3,"is_active":false}]`), "json", categoryColumns, []string{"name", "slug"})
	if len(errs) != 0 || len(rows) != 1 || rows[0].get("sort_order") != "3" || rows[0].get("is_active") != "false" {
		t.Fatalf("rows=%+v errs=%+v", rows, errs)
	}
	if _, errs = ParseRows([]byte(`{"name":"x"}`), "json", categoryColumns, nil); len(errs) == 0 {
		t.Fatal("object instead of array must fail")
	}
}

func TestImportCategoriesParentInSameFileAnyOrder(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	rows := parseCSV(t, catHeader+"Con,con,,cha,2,true,,\nCha,cha,Mô tả,,1,true,https://x/c.jpg,Ảnh\n", categoryColumns, []string{"name", "slug"})
	dry, err := svc.ImportCategories(context.Background(), rows, false)
	if err != nil || len(dry.Errors) != 0 || dry.Created != 2 || dry.Committed || len(repo.cats) != 0 {
		t.Fatalf("dry run must not write: %+v err=%v cats=%d", dry, err, len(repo.cats))
	}
	got, err := svc.ImportCategories(context.Background(), rows, true)
	if err != nil || !got.Committed || got.Created != 2 || len(repo.cats) != 2 {
		t.Fatalf("commit: %+v err=%v", got, err)
	}
	if repo.log[0] != "create-category cha" || repo.log[1] != "create-category con" {
		t.Fatalf("parent must be created first: %v", repo.log)
	}
	if repo.cats[1]["parent_id"] != repo.cats[0]["id"] {
		t.Fatalf("child parent_id = %v, want %v", repo.cats[1]["parent_id"], repo.cats[0]["id"])
	}
	// Re-importing the same file changes nothing.
	again, err := svc.ImportCategories(context.Background(), rows, true)
	if err != nil || again.Created != 0 || again.Updated != 0 || again.Skipped != 2 {
		t.Fatalf("idempotent re-import: %+v", again)
	}
	// Editing a name updates with the current version.
	rows = parseCSV(t, catHeader+"Cha mới,cha,Mô tả,,1,true,https://x/c.jpg,Ảnh\n", categoryColumns, []string{"name", "slug"})
	upd, err := svc.ImportCategories(context.Background(), rows, true)
	if err != nil || upd.Updated != 1 || repo.cats[0]["name"] != "Cha mới" || repo.cats[0]["version"].(int64) != 1 {
		t.Fatalf("update: %+v err=%v cat=%v", upd, err, repo.cats[0])
	}
}

func TestImportCategoriesRowErrors(t *testing.T) {
	repo := &memRepo{}
	svc := NewService(repo)
	rows := parseCSV(t, catHeader+"Aa,Bad Slug,,,,,,\nBb,b,,khong-co,,,,\nCc,c,,c,,,,\nDd,d,,,x,,,\nEe,e,,,,maybe,,\nFf,f,,,,,,\nF2,f,,,,,,\nGg,g,,h,,,,\nHh,h,,g,,,,\n", categoryColumns, []string{"name", "slug"})
	got, err := svc.ImportCategories(context.Background(), rows, true)
	if err != nil || got.Committed || len(repo.cats) != 0 {
		t.Fatalf("errors must block commit: %+v", got)
	}
	want := map[int]string{2: "slug", 3: "parent_slug", 4: "parent_slug", 5: "sort_order", 6: "is_active", 8: "slug", 9: "parent_slug", 10: "parent_slug"}
	have := map[int]string{}
	for _, e := range got.Errors {
		have[e.Row] = e.Field
	}
	for row, field := range want {
		if have[row] != field {
			t.Errorf("row %d: field %q, want %q (errors %+v)", row, have[row], field, got.Errors)
		}
	}
}

func seedCategory(repo *memRepo, slug string) string {
	c, _ := repo.CreateCategory(context.Background(), CategoryInput{Name: slug, Slug: slug, IsActive: true})
	repo.log = nil
	return c["id"].(string)
}

func TestImportProductsGroupsRowsBySlugAndValidates(t *testing.T) {
	repo := &memRepo{}
	seedCategory(repo, "ao-dai")
	svc := NewService(repo)
	csvText := prodHeader +
		"ao-a,Áo A,ao-dai,Mô tả,Lụa,1500000,1800000,true,https://x/a.jpg,https://x/1.jpg|https://x/2.jpg,Đỏ,#ff0000,s,AO-A-DO-S,5\n" +
		"ao-a,,,,,,,,,,Đỏ,#ff0000,M,AO-A-DO-M,3\n" +
		"ao-a,,,,,,,,,,Xanh,,freesize,AO-A-XANH-F,0\n" +
		"ao-b,Áo B,ao-dai,,,900000,,false,,,,,,AO-B,7\n"
	rows := parseCSV(t, csvText, productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	// ao-b has no colour: validateVariant requires one.
	got, err := svc.ImportProducts(context.Background(), rows, true)
	if err != nil || got.Committed || len(got.Errors) != 1 || got.Errors[0].Row != 5 || got.Errors[0].Field != "color" {
		t.Fatalf("want a single colour error on row 5: %+v err=%v", got, err)
	}
	if len(repo.products) != 0 {
		t.Fatal("nothing may be written when any row has an error")
	}
	rows = parseCSV(t, strings.Replace(csvText, "ao-b,Áo B,ao-dai,,,900000,,false,,,,,,AO-B,7", "ao-b,Áo B,ao-dai,,,900000,,false,,,Trắng,,,AO-B,7", 1), productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	got, err = svc.ImportProducts(context.Background(), rows, true)
	if err != nil || !got.Committed || got.Created != 2 || len(got.Errors) != 0 {
		t.Fatalf("commit: %+v err=%v", got, err)
	}
	a := repo.products[0]
	variants := a["variants"].([]any)
	if len(variants) != 3 {
		t.Fatalf("rows with the same product_slug become one product with 3 SKUs: %v", variants)
	}
	sizes := []string{}
	for _, v := range variants {
		sizes = append(sizes, v.(map[string]any)["size"].(string))
	}
	if strings.Join(sizes, ",") != "S,M,Freesize" {
		t.Fatalf("sizes must be stored in standard spelling: %v", sizes)
	}
	if a["price"].(int64) != 1500000 || a["compare_at_price"].(int64) != 1800000 || len(a["colors"].([]any)) != 2 {
		t.Fatalf("product fields: %+v", a)
	}
}

func TestImportProductsRowErrors(t *testing.T) {
	repo := &memRepo{}
	seedCategory(repo, "ao-dai")
	svc := NewService(repo)
	rows := parseCSV(t, prodHeader+
		"p1,Áo thử,ao-dai,,,100000,,,,,Đỏ,,XXXL,P1-A,1\n"+ // 2 bad size
		"p2,Áo thử,khong-co,,,100000,,,,,Đỏ,,M,P2-A,1\n"+ // 3 unknown category
		"p3,Áo thử,ao-dai,,,abc,,,,,Đỏ,,M,P3-A,1\n"+ // 4 bad price
		"p4,Áo thử,ao-dai,,,100000,,,,,Đỏ,red,M,P4-A,1\n"+ // 5 bad hex
		"p5,Áo thử,ao-dai,,,100000,,,,,Đỏ,,M,P5-A,-2\n"+ // 6 bad stock
		"p6,Áo thử,ao-dai,,,100000,,,,,Đỏ,,M,DUP,1\n"+ // 7 ok
		"p7,Áo thử,ao-dai,,,100000,,,,,Đỏ,,M,dup,1\n"+ // 8 duplicate SKU
		"p6,Khác,ao-dai,,,100000,,,,,Xanh,,M,P6-B,1\n"+ // 9 contradicts name
		"p8,Áo thử,ao-dai,,,100000,,,,,Đỏ,,M,P8-A,1\np8,,,,,,,,,,Đỏ,,M,P8-B,1\n", // 10/11 duplicate colour+size
		productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	got, err := svc.ImportProducts(context.Background(), rows, false)
	if err != nil {
		t.Fatal(err)
	}
	have := map[int]string{}
	for _, e := range got.Errors {
		have[e.Row] = e.Field
	}
	for row, field := range map[int]string{2: "size", 3: "category_slug", 4: "price", 5: "color_hex", 6: "stock", 8: "sku", 9: "name", 11: "color_name"} {
		if have[row] != field {
			t.Errorf("row %d: field %q, want %q (all: %+v)", row, have[row], field, got.Errors)
		}
	}
	if _, ok := have[7]; ok {
		t.Errorf("row 7 is valid: %+v", got.Errors)
	}
}

func TestImportProductsNeverOverwritesExistingStock(t *testing.T) {
	repo := &memRepo{}
	cat := seedCategory(repo, "ao-dai")
	svc := NewService(repo)
	_, _ = repo.CreateProduct(context.Background(), ProductCreate{ProductInput: ProductInput{Name: "Áo A", Slug: "ao-a", CategoryID: cat, Price: 100000, IsActive: true},
		Variants: []Variant{{SKU: "AO-A-DO-S", Color: "Đỏ", Size: "S", Stock: 5}}})
	repo.log = nil
	rows := parseCSV(t, prodHeader+
		"ao-a,Áo A (mới),ao-dai,,,200000,,true,,,Đỏ,,S,AO-A-DO-S,999\n"+
		"ao-a,,,,,,,,,,Đỏ,,M,AO-A-DO-M,4\n", productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	dry, err := svc.ImportProducts(context.Background(), rows, false)
	if err != nil || len(dry.Errors) != 0 || dry.Updated != 1 || dry.IgnoredStock != 1 || len(repo.log) != 0 {
		t.Fatalf("dry run: %+v err=%v log=%v", dry, err, repo.log)
	}
	got, err := svc.ImportProducts(context.Background(), rows, true)
	if err != nil || !got.Committed || got.Updated != 1 || got.Created != 0 {
		t.Fatalf("commit: %+v err=%v", got, err)
	}
	p := repo.products[0]
	byStock := map[string]int64{}
	for _, v := range p["variants"].([]any) {
		m := v.(map[string]any)
		byStock[m["sku"].(string)] = m["stock"].(int64)
	}
	if byStock["AO-A-DO-S"] != 5 {
		t.Fatalf("existing SKU stock must stay 5, got %d", byStock["AO-A-DO-S"])
	}
	if byStock["AO-A-DO-M"] != 4 {
		t.Fatalf("new SKU opening stock is applied as a delta: %d", byStock["AO-A-DO-M"])
	}
	if p["version"].(int64) != 1 || p["name"] != "Áo A (mới)" || p["price"].(int64) != 200000 {
		t.Fatalf("product update must bump version: %+v", p)
	}
	joined := strings.Join(repo.log, "|")
	if !strings.Contains(joined, "add-variant AO-A-DO-M") || !strings.Contains(joined, "adjust AO-A-DO-M +4") || strings.Contains(joined, "adjust AO-A-DO-S") {
		t.Fatalf("log = %v", repo.log)
	}
	if len(got.Notes) == 0 || !strings.Contains(got.Notes[0], "bỏ qua") {
		t.Fatalf("notes must mention ignored stock: %v", got.Notes)
	}
	// Same file again: nothing left to do except the ignored stock note.
	again, err := svc.ImportProducts(context.Background(), rows, true)
	if err != nil || again.Updated != 0 || again.Skipped != 1 {
		t.Fatalf("idempotent: %+v err=%v", again, err)
	}
}

func TestImportProductsRejectsSKUOwnedByAnotherProduct(t *testing.T) {
	repo := &memRepo{}
	cat := seedCategory(repo, "ao-dai")
	svc := NewService(repo)
	_, _ = repo.CreateProduct(context.Background(), ProductCreate{ProductInput: ProductInput{Name: "Khác", Slug: "khac", CategoryID: cat, Price: 1000, IsActive: true},
		Variants: []Variant{{SKU: "TAKEN", Color: "Đỏ", Size: "S"}}})
	rows := parseCSV(t, prodHeader+"moi,Mới,ao-dai,,,100000,,,,,Đỏ,,S,taken,1\n", productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	got, err := svc.ImportProducts(context.Background(), rows, true)
	if err != nil || got.Committed || len(got.Errors) != 1 || got.Errors[0].Field != "sku" || !strings.Contains(got.Errors[0].Message, "khac") {
		t.Fatalf("%+v err=%v", got, err)
	}
}

func TestExportThenImportRoundTrips(t *testing.T) {
	repo := &memRepo{}
	cat := seedCategory(repo, "ao-dai")
	svc := NewService(repo)
	_, _ = repo.CreateProduct(context.Background(), ProductCreate{
		ProductInput: ProductInput{Name: "Áo A", Slug: "ao-a", CategoryID: cat, Price: 100000, IsActive: true, Colors: []Color{{Name: "Đỏ", Hex: "#ff0000"}}},
		Variants:     []Variant{{SKU: "A-1", Color: "Đỏ", Size: "S", Stock: 3}, {SKU: "A-2", Color: "Đỏ", Size: "M", Stock: 0}},
	})
	exported, err := svc.ExportProducts(context.Background())
	if err != nil || len(exported) != 3 || exported[0][0] != "product_slug" {
		t.Fatalf("export: %v err=%v", exported, err)
	}
	var sb strings.Builder
	for _, line := range exported {
		sb.WriteString(strings.Join(line, ",") + "\n")
	}
	rows := parseCSV(t, sb.String(), productColumns, []string{"product_slug", "name", "category_slug", "price", "sku"})
	got, err := svc.ImportProducts(context.Background(), rows, false)
	if err != nil || len(got.Errors) != 0 || got.Created != 0 || got.Updated != 0 || got.Skipped != 1 {
		t.Fatalf("exported file must re-import as a no-op: %+v err=%v", got, err)
	}
	cats, err := svc.ExportCategories(context.Background())
	if err != nil || len(cats) != 2 || cats[1][1] != "ao-dai" {
		t.Fatalf("categories export: %v err=%v", cats, err)
	}
}
