package main

import (
	"strings"
	"testing"
	"time"

	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func fixture(t *testing.T, cod *int64) (options, batch) {
	t.Helper()
	wards, err := loadWards("../../../client/src/data/vn-divisions-2025.json")
	if err != nil {
		t.Fatal(err)
	}
	products := []product{
		{ID: bson.NewObjectID(), Name: "Áo dài lụa Phúc Thọ", Price: 1_250_000, Variants: []variant{{"A1S", "S", "Đỏ"}, {"A1M", "M", "Đỏ"}}},
		{ID: bson.NewObjectID(), Name: "Áo dài tà đôi Vườn Hoa", Price: 880_000, Variants: []variant{{"A2XS", "XS", "Hồng"}, {"A2L", "L", "Hồng"}}},
		{ID: bson.NewObjectID(), Name: "Áo dài Tết Quý Xuân", Price: 990_000, Variants: []variant{{"A3M", "M", "Xanh"}}},
	}
	from := time.Date(2026, 11, 21, 0, 0, 0, 0, vnZone)
	o := options{From: from, Now: time.Date(2026, 11, 28, 1, 30, 0, 0, vnZone), Seed: 7, Customers: 40, CODFee: cod}
	b, err := generate(o, products, wards)
	if err != nil {
		t.Fatal(err)
	}
	return o, b
}

func TestGeneratedHistoryIsPlausible(t *testing.T) {
	fee := int64(30_000)
	o, b := fixture(t, &fee)
	if len(b.Orders) < 40 || len(b.Orders) > 90 {
		t.Fatalf("a week should have roughly 6–12 orders a day, got %d", len(b.Orders))
	}
	joined := map[bson.ObjectID]time.Time{}
	for _, c := range b.Customers {
		joined[c["_id"].(bson.ObjectID)] = c["created_at"].(time.Time)
		name := c["full_name"].(string)
		if len(strings.Fields(name)) != 3 || !strings.HasSuffix(c["email"].(string), "@example.com") {
			t.Fatalf("customer %q / %v", name, c["email"])
		}
	}
	statuses := map[string]int{}
	numbers := map[string]bool{}
	for _, d := range b.Orders {
		created, updated := d["created_at"].(time.Time), d["updated_at"].(time.Time)
		if created.Before(o.From) || !created.Before(o.Now) || updated.Before(created) || updated.After(o.Now) {
			t.Fatalf("order times out of range: created %s updated %s", created, updated)
		}
		if h := created.In(vnZone).Hour(); h >= 1 && h <= 4 {
			// Possible but rare; more than a couple would mean the hour weights are off.
			statuses["night"]++
		}
		if id, ok := d["customer_id"].(bson.ObjectID); ok && joined[id].After(created) {
			t.Fatalf("customer ordered before signing up")
		}
		number := d["order_number"].(string)
		if numbers[number] || !strings.HasPrefix(number, "AD-"+created.UTC().Format("060102")) {
			t.Fatalf("order number %q", number)
		}
		numbers[number] = true
		var sum int64
		for _, it := range d["items"].(bson.A) {
			item := it.(bson.M)
			sum += item["unit_price"].(int64) * int64(item["quantity"].(int))
		}
		method, status, payment := d["payment_method"].(string), d["status"].(string), d["payment_status"].(string)
		wantFee := int64(0)
		if method == domainorder.PaymentMethodCOD {
			wantFee = fee
		}
		if d["subtotal"] != sum || d["shipping_fee"] != wantFee || d["total"] != sum+wantFee {
			t.Fatalf("money mismatch: %v", d)
		}
		if method == domainorder.PaymentMethodCashAtCounter && status == domainorder.StatusShipping {
			t.Fatal("counter orders are never shipped")
		}
		if (status == domainorder.StatusDelivered) != (payment == domainorder.PaymentPaid) {
			t.Fatalf("delivered orders are paid and only those: %s/%s", status, payment)
		}
		if _, ok := d["cancelled_at"]; ok != (status == domainorder.StatusCancelled) {
			t.Fatal("cancelled_at must match cancelled status")
		}
		if d["seed_tag"] != seedTag {
			t.Fatal("missing seed tag")
		}
		statuses[status]++
	}
	if statuses["night"] > 3 {
		t.Fatalf("too many orders between 1h and 4h: %d", statuses["night"])
	}
	for _, s := range []string{domainorder.StatusPending, domainorder.StatusConfirmed, domainorder.StatusShipping, domainorder.StatusDelivered, domainorder.StatusCancelled} {
		if statuses[s] == 0 {
			t.Fatalf("expected some %s orders: %v", s, statuses)
		}
	}
	t.Logf("%d customers, %d orders, statuses %v", len(b.Customers), len(b.Orders), statuses)
}

func TestNoCODFeeMeansCounterOrdersOnly(t *testing.T) {
	_, b := fixture(t, nil)
	for _, d := range b.Orders {
		if d["payment_method"] != domainorder.PaymentMethodCashAtCounter || d["shipping_fee"] != int64(0) {
			t.Fatalf("COD must not be generated without ORDER_COD_SHIPPING_FEE: %v", d["payment_method"])
		}
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	_, a := fixture(t, nil)
	_, b := fixture(t, nil)
	if len(a.Orders) != len(b.Orders) || a.Customers[0]["full_name"] != b.Customers[0]["full_name"] || a.Orders[0]["total"] != b.Orders[0]["total"] {
		t.Fatal("same seed must give the same history")
	}
}

func TestASCIISlug(t *testing.T) {
	if got := asciiSlug("Đặng Thị Ánh Tuyết", "."); got != "dang.thi.anh.tuyet" {
		t.Fatalf("got %q", got)
	}
}
