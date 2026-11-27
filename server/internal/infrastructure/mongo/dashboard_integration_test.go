//go:build integration

package mongo

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	app "github.com/your-org/aodora/server/internal/application/dashboard"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// scratchDatabase connects to MONGO_TEST_URI and returns a throw-away database (dropped on cleanup).
func scratchDatabase(t *testing.T, prefix string) *driver.Database {
	t.Helper()
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI is required for integration tests")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + prefix + bson.NewObjectID().Hex()
	client, err := driver.Connect(options.Client().ApplyURI(u.String()))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(strings.TrimPrefix(u.Path, "/"))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = db.Drop(ctx)
		_ = client.Disconnect(ctx)
	})
	return db
}

func dec(t *testing.T, s string) bson.Decimal128 {
	t.Helper()
	d, err := bson.ParseDecimal128(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func vnTime(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, time.FixedZone("ICT", 7*3600)).UTC()
}

func TestDashboardAggregations(t *testing.T) {
	db := scratchDatabase(t, "aodora_dash_test_")
	ctx := context.Background()
	catA, catB := bson.NewObjectID(), bson.NewObjectID()
	p1, p2, p3 := bson.NewObjectID(), bson.NewObjectID(), bson.NewObjectID()
	// Make ids ordered so ties sort deterministically.
	if p1.Hex() > p2.Hex() {
		p1, p2 = p2, p1
	}
	must := func(_ any, err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(db.Collection("categories").InsertMany(ctx, []any{bson.M{"_id": catA, "name": "Áo nam", "slug": "ao-nam"}, bson.M{"_id": catB, "name": "Áo nữ", "slug": "ao-nu"}}))
	must(db.Collection("products").InsertMany(ctx, []any{
		bson.M{"_id": p1, "name": "Áo A", "slug": "ao-a", "category_id": catA, "price": dec(t, "200000"), "is_active": true, "stock": 7, "image": bson.M{"src": "http://x/a.jpg"},
			"variants": bson.A{bson.M{"sku": "A-M", "size": "M", "stock": 2}, bson.M{"sku": "A-L", "size": "L", "stock": 5}}},
		bson.M{"_id": p2, "name": "Áo B", "slug": "ao-b", "category_id": catB, "price": int64(100000), "is_active": true, "stock": 3, "images": bson.A{"http://x/b.jpg"},
			"variants": bson.A{bson.M{"sku": "B-L", "size": "L", "stock": 0}, bson.M{"sku": "B-XL", "size": "XL", "stock": 3}}},
		bson.M{"_id": p3, "name": "Áo C ẩn", "slug": "ao-c", "category_id": catA, "price": int32(50000), "is_active": false, "stock": 10,
			"variants": bson.A{bson.M{"sku": "C-M", "size": "M", "stock": 0}}},
	}))
	items := func(parts ...any) bson.A { return bson.A(parts) }
	item := func(p bson.ObjectID, name string, qty int32, unit any, size string) bson.M {
		return bson.M{"product_id": p, "product_name": name, "sku": name + size, "quantity": qty, "unit_price": unit, "size": size}
	}
	order := func(at time.Time, status, pay, method string, total any, its bson.A) bson.M {
		return bson.M{"status": status, "payment_status": pay, "payment_method": method, "total": total, "items": its, "created_at": at}
	}
	must(db.Collection("orders").InsertMany(ctx, []any{
		// o1/o2: same local day (03-02), o2 at 23:30 local = 16:30 UTC.
		order(vnTime(2026, 3, 2, 10, 0), "delivered", "paid", "cod", dec(t, "500000"), items(item(p1, "Áo A", 2, dec(t, "200000"), "M"), item(p2, "Áo B", 1, int64(100000), "L"))),
		order(vnTime(2026, 3, 2, 23, 30), "shipping", "paid", "cod", int64(300000), items(item(p1, "Áo A", 1, int64(200000), "M"), item(p2, "Áo B", 1, dec(t, "100000"), "L"))),
		// 00:30 local on 03-03 is still 03-02 in UTC: must be bucketed on 03-03.
		order(vnTime(2026, 3, 3, 0, 30), "pending", "unpaid", "cash_at_counter", dec(t, "1000000"), items(item(p1, "Áo A", 5, dec(t, "200000"), "M"))),
		order(vnTime(2026, 3, 5, 9, 0), "cancelled", "paid", "cod", dec(t, "700000"), items(item(p2, "Áo B", 7, dec(t, "100000"), "L"))),
		order(vnTime(2026, 3, 10, 23, 59), "delivered", "paid", "cash_at_counter", dec(t, "100000"), items(item(p2, "Áo B", 1, dec(t, "100000"), "L"))),
		// outside the window (exactly the next local midnight) and in the previous period.
		order(vnTime(2026, 3, 11, 0, 0), "delivered", "paid", "cod", dec(t, "999999"), items(item(p1, "Áo A", 1, dec(t, "999999"), "M"))),
		order(vnTime(2026, 2, 20, 12, 0), "delivered", "paid", "cod", dec(t, "400000"), items(item(p1, "Áo A", 2, dec(t, "200000"), "M"))),
		order(vnTime(2026, 2, 21, 12, 0), "cancelled", "unpaid", "cod", dec(t, "123"), items(item(p2, "Áo B", 1, dec(t, "123"), "L"))),
	}))
	must(db.Collection("customers").InsertMany(ctx, []any{
		bson.M{"full_name": "a", "created_at": vnTime(2026, 3, 2, 8, 0)}, bson.M{"full_name": "b", "created_at": vnTime(2026, 3, 4, 8, 0)},
		bson.M{"full_name": "c", "created_at": vnTime(2026, 3, 11, 0, 0)}, bson.M{"full_name": "d", "created_at": vnTime(2026, 2, 25, 8, 0)},
	}))
	must(db.Collection("tailoring_requests").InsertMany(ctx, []any{
		bson.M{"status": "submitted", "created_at": vnTime(2026, 3, 2, 8, 0)}, bson.M{"status": "submitted", "created_at": vnTime(2026, 3, 3, 8, 0)},
		bson.M{"status": "completed", "created_at": vnTime(2026, 3, 4, 8, 0)}, bson.M{"status": "submitted", "created_at": vnTime(2026, 2, 4, 8, 0)},
	}))

	repo := NewDashboardRepository(db)
	w, err := app.ResolveWindow(time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC), "2026-03-01", "2026-03-10", "day", 3)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := repo.Aggregate(ctx, w)
	if err != nil {
		t.Fatal(err)
	}
	if raw.Current != (app.Period{Orders: 5, Cancelled: 1, PaidOrders: 3, Revenue: 900000, NewCustomers: 2}) {
		t.Fatalf("current = %+v", raw.Current)
	}
	if raw.Previous != (app.Period{Orders: 2, Cancelled: 1, PaidOrders: 1, Revenue: 400000, NewCustomers: 1}) {
		t.Fatalf("previous = %+v", raw.Previous)
	}
	report := app.Shape(w, raw)
	if len(report.Series) != 10 {
		t.Fatalf("series len = %d", len(report.Series))
	}
	byDate := map[string]app.Point{}
	for _, p := range report.Series {
		byDate[p.Date] = p
	}
	for date, want := range map[string]app.Point{
		"2026-03-02": {Date: "2026-03-02", Revenue: 800000, Orders: 2},
		"2026-03-03": {Date: "2026-03-03", Revenue: 0, Orders: 1},
		"2026-03-05": {Date: "2026-03-05", Revenue: 0, Orders: 1},
		"2026-03-10": {Date: "2026-03-10", Revenue: 100000, Orders: 1},
		"2026-03-04": {Date: "2026-03-04"},
	} {
		if byDate[date] != want {
			t.Fatalf("%s = %+v, want %+v", date, byDate[date], want)
		}
	}
	counts := func(in []app.Count) map[string]int64 {
		out := map[string]int64{}
		for _, c := range in {
			out[c.Key] = c.Count
		}
		return out
	}
	if got := counts(raw.ByStatus); got["delivered"] != 2 || got["shipping"] != 1 || got["pending"] != 1 || got["cancelled"] != 1 || len(got) != 4 {
		t.Fatalf("by status = %v", got)
	}
	if got := counts(raw.ByPaymentStatus); got["paid"] != 4 || got["unpaid"] != 1 {
		t.Fatalf("by payment status = %v", got)
	}
	methods := map[string]app.Count{}
	for _, c := range raw.ByPaymentMethod {
		methods[c.Key] = c
	}
	if methods["cod"].Count != 3 || methods["cod"].Revenue != 800000 || methods["cash_at_counter"].Count != 2 || methods["cash_at_counter"].Revenue != 100000 {
		t.Fatalf("by method = %+v", methods)
	}
	if len(raw.TopRevenue) != 2 || raw.TopRevenue[0].ProductID != p1.Hex() || raw.TopRevenue[0].Revenue != 600000 || raw.TopRevenue[0].Quantity != 3 || raw.TopRevenue[0].Name != "Áo A" || raw.TopRevenue[0].Slug != "ao-a" || raw.TopRevenue[0].Image != "http://x/a.jpg" {
		t.Fatalf("top revenue = %+v", raw.TopRevenue)
	}
	if raw.TopRevenue[1].Revenue != 300000 || raw.TopRevenue[1].Image != "http://x/b.jpg" {
		t.Fatalf("top revenue[1] = %+v", raw.TopRevenue[1])
	}
	if len(raw.TopQuantity) != 2 || raw.TopQuantity[0].ProductID != p1.Hex() {
		t.Fatalf("top quantity (tie broken by id) = %+v", raw.TopQuantity)
	}
	if len(raw.ByCategory) != 2 || raw.ByCategory[0].Name != "Áo nam" || raw.ByCategory[0].Revenue != 600000 || raw.ByCategory[1].Name != "Áo nữ" || raw.ByCategory[1].Revenue != 300000 {
		t.Fatalf("by category = %+v", raw.ByCategory)
	}
	if len(raw.BySize) != 2 || raw.BySize[0].Size != "M" || raw.BySize[0].Revenue != 600000 || raw.BySize[0].Quantity != 3 || raw.BySize[1].Size != "L" || raw.BySize[1].Revenue != 300000 {
		t.Fatalf("by size = %+v", raw.BySize)
	}
	if got := counts(raw.Tailoring); got["submitted"] != 2 || got["completed"] != 1 || len(got) != 2 {
		t.Fatalf("tailoring = %v", got)
	}
	inv := raw.Inventory
	if inv.ActiveProducts != 2 || inv.HiddenProducts != 1 || inv.StockValue != 2_200_000 || inv.OutOfStockSKUs != 1 || inv.LowStockSKUs != 3 || inv.LowStockThreshold != 3 {
		t.Fatalf("inventory = %+v", inv)
	}
	if len(inv.LowStock) != 3 || inv.LowStock[0].SKU != "B-L" || inv.LowStock[0].Stock != 0 || inv.LowStock[1].SKU != "A-M" || inv.LowStock[2].SKU != "B-XL" {
		t.Fatalf("low stock list = %+v", inv.LowStock)
	}

	// Week and month granularities group on local Monday / first of month.
	w.Granularity = app.Week
	raw, err = repo.Aggregate(ctx, w)
	if err != nil {
		t.Fatal(err)
	}
	weeks := app.FillSeries(w.Start, w.End, app.Week, raw.Series)
	// 2026-03-01 is a Sunday: weeks start 02-23, 03-02, 03-09.
	if len(weeks) != 3 || weeks[0].Date != "2026-02-23" || weeks[1].Date != "2026-03-02" || weeks[1].Orders != 4 || weeks[1].Revenue != 800000 || weeks[2].Orders != 1 || weeks[0].Orders != 0 {
		t.Fatalf("weeks = %+v", weeks)
	}
}
