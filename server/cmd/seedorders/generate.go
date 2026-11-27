package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
	"unicode"

	apporder "github.com/your-org/aodora/server/internal/application/order"
	domainorder "github.com/your-org/aodora/server/internal/domain/order"
	"go.mongodb.org/mongo-driver/v2/bson"
	"golang.org/x/text/unicode/norm"
)

// Every seeded document carries this tag so a rerun replaces the batch
// instead of piling up, and -purge removes it without touching real data.
const (
	seedTag       = "order-history"
	checkoutScope = "seed-order-history"
)

var vnZone = time.FixedZone("ICT", 7*3600)

type variant struct{ SKU, Size, Color string }

type product struct {
	ID       bson.ObjectID
	Name     string
	Price    int64
	Variants []variant
}

type ward struct{ Name, Province string }

type customer struct {
	ID       bson.ObjectID
	Name     string
	Email    string
	Phone    string
	Address  address
	JoinedAt time.Time
}

type address struct{ Line1, Ward, City string }

type options struct {
	From, Now time.Time
	Seed      uint64
	Customers int
	// CODFee is ORDER_COD_SHIPPING_FEE; nil means COD is unavailable (Q-04), so
	// only cash-at-counter orders are generated.
	CODFee *int64
}

type batch struct {
	Customers []bson.M
	Orders    []bson.M
}

var (
	familyNames  = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng", "Bùi", "Đỗ", "Hồ", "Ngô", "Dương", "Lý", "Đinh", "Trịnh", "Mai", "Cao"}
	femaleMiddle = []string{"Thị", "Ngọc", "Thu", "Thanh", "Minh", "Phương", "Bảo", "Hoài", "Khánh", "Mỹ", "Diệu", "Quỳnh", "Thảo", "Hồng", "Kim"}
	femaleGiven  = []string{"Anh", "Trang", "Linh", "Hà", "Hương", "Lan", "Mai", "Ngân", "Nhung", "Thảo", "Vy", "Yến", "Hạnh", "Tâm", "Uyên", "Châu", "Giang", "Hiền", "My", "Nhi", "Oanh", "Phương", "Quyên", "Trâm", "Duyên", "Thư", "Tiên", "Loan", "Hằng", "Diệp"}
	maleMiddle   = []string{"Văn", "Minh", "Quốc", "Đức", "Thành", "Hoàng", "Gia", "Hữu"}
	maleGiven    = []string{"Huy", "Nam", "Khoa", "Tuấn", "Long", "Phúc", "Bảo", "Dũng", "Hải", "Sơn"}
	streets      = []string{"Lê Lợi", "Trần Hưng Đạo", "Nguyễn Trãi", "Hai Bà Trưng", "Lý Thường Kiệt", "Phan Đình Phùng", "Nguyễn Du", "Hùng Vương", "Điện Biên Phủ", "Lê Duẩn", "Bà Triệu", "Nguyễn Huệ", "Hoàng Diệu", "Trần Phú", "Quang Trung", "Võ Thị Sáu", "Nguyễn Thị Minh Khai", "Pasteur", "Cách Mạng Tháng Tám", "Lạc Long Quân"}
	notes        = []string{"Giao giờ hành chính giúp mình", "Gọi trước khi giao 15 phút", "Gói quà giúp mình, không kèm hoá đơn", "Áo mặc dịp cưới, mong shop giao sớm", "Giao buổi tối sau 18h", "Mình cần trước Chủ nhật"}
	// Demand for the shop's size range; skewed to S/M like Vietnamese sizing.
	sizeWeight = map[string]float64{"XS": 0.8, "S": 2.2, "M": 2.6, "L": 1.6, "XL": 0.7}
	// Hourly order weight in Vietnam time: lunch break and evening peaks.
	hourWeight = [24]float64{0.1, 0.05, 0.02, 0.01, 0.01, 0.05, 0.2, 0.5, 0.9, 1.1, 1.2, 1.6, 1.8, 1.3, 1.0, 1.0, 1.0, 1.1, 1.3, 1.8, 2.2, 2.4, 1.9, 0.9}
	// Orders per weekday (Sunday first); weekends and Friday evening sell more.
	dayBase = [7]float64{11, 6, 6, 7, 7, 9, 12}
)

// generate builds customers and orders between From and Now. It is pure and
// deterministic for a given seed so tests can check the invariants.
func generate(o options, products []product, wards []ward) (batch, error) {
	if len(products) == 0 {
		return batch{}, fmt.Errorf("no sellable products with variants: import the catalog first")
	}
	if len(wards) == 0 {
		return batch{}, fmt.Errorf("no wards loaded")
	}
	if !o.From.Before(o.Now) {
		return batch{}, fmt.Errorf("from %s must be before now %s", o.From, o.Now)
	}
	r := rand.New(rand.NewPCG(o.Seed, o.Seed^0x9e3779b97f4a7c15))
	popularity := make([]float64, len(products))
	for i := range products {
		// Zipf-like: a handful of best sellers, a long tail.
		popularity[i] = 1 / float64(1+r.IntN(len(products)))
	}

	var out batch
	customers := make([]customer, o.Customers)
	usedEmail := map[string]int{}
	for i := range customers {
		name := vietnameseName(r)
		local := asciiSlug(name, ".")
		usedEmail[local]++
		if n := usedEmail[local]; n > 1 {
			local = fmt.Sprintf("%s%d", local, n)
		}
		joined := o.From.Add(-time.Duration(r.IntN(75*24)) * time.Hour)
		if r.Float64() < 0.35 {
			// New sign-ups inside the window feed the "khách mới" KPI.
			joined = randomTime(r, o.From, o.Now)
		}
		c := customer{ID: bson.NewObjectID(), Name: name, Email: local + "@example.com", Phone: phone(r), Address: randomAddress(r, wards), JoinedAt: joined}
		customers[i] = c
		out.Customers = append(out.Customers, bson.M{
			"_id": c.ID, "full_name": c.Name, "email": c.Email, "email_normalized": c.Email, "phone": c.Phone, "role": "customer",
			"addresses":  bson.A{bson.M{"recipient": c.Name, "phone": c.Phone, "line1": c.Address.Line1, "ward": c.Address.Ward, "city": c.Address.City, "is_default": true}},
			"created_at": c.JoinedAt.UTC(), "updated_at": c.JoinedAt.UTC(), "seed_tag": seedTag,
		})
	}

	usedNumbers := map[string]bool{}
	for day := startOfDay(o.From); day.Before(o.Now); day = day.AddDate(0, 0, 1) {
		count := int(dayBase[day.Weekday()]*(0.75+0.5*r.Float64()) + 0.5)
		for k := 0; k < count; k++ {
			created := day.Add(time.Duration(pickHour(r))*time.Hour + time.Duration(r.IntN(3600))*time.Second)
			if created.Before(o.From) || !created.Before(o.Now) {
				continue
			}
			var buyer *customer
			if r.Float64() < 0.8 {
				eligible := make([]int, 0, len(customers))
				for i := range customers {
					if !customers[i].JoinedAt.After(created) {
						eligible = append(eligible, i)
					}
				}
				if len(eligible) > 0 {
					buyer = &customers[eligible[r.IntN(len(eligible))]]
				}
			}
			number := apporder.DefaultOrderNumber(created)
			for usedNumbers[number] {
				number = apporder.DefaultOrderNumber(created)
			}
			usedNumbers[number] = true
			out.Orders = append(out.Orders, buildOrder(r, o, created, number, buyer, products, popularity, wards))
		}
	}
	return out, nil
}

func buildOrder(r *rand.Rand, o options, created time.Time, number string, buyer *customer, products []product, popularity []float64, wards []ward) bson.M {
	method := domainorder.PaymentMethodCashAtCounter
	if o.CODFee != nil && r.Float64() < 0.78 {
		method = domainorder.PaymentMethodCOD
	}
	lineCount := 1
	if x := r.Float64(); x > 0.97 {
		lineCount = 3
	} else if x > 0.75 {
		lineCount = 2
	}
	items := bson.A{}
	seen := map[string]bool{}
	var subtotal int64
	for len(items) < lineCount {
		p := products[weighted(r, popularity)]
		v := pickVariant(r, p.Variants)
		if seen[v.SKU] {
			if len(seen) >= len(products) {
				break
			}
			continue
		}
		seen[v.SKU] = true
		qty := 1
		if r.Float64() < 0.1 {
			qty = 2
		}
		item := bson.M{"product_id": p.ID, "product_name": p.Name, "sku": v.SKU, "quantity": qty, "unit_price": p.Price, "fulfillment_mode": "ready"}
		if v.Size != "" {
			item["size"] = v.Size
		}
		if v.Color != "" {
			item["color"] = v.Color
		}
		items = append(items, item)
		subtotal += p.Price * int64(qty)
	}
	var fee int64
	if method == domainorder.PaymentMethodCOD {
		fee = *o.CODFee
	}

	status, payment, updated, paidAt, cancelledAt := lifecycle(r, method, created, o.Now)
	recipient, tel, addr := "", "", randomAddress(r, wards)
	doc := bson.M{
		"_id": bson.NewObjectID(), "order_number": number, "status": status, "payment_method": method, "payment_status": payment,
		"subtotal": subtotal, "shipping_fee": fee, "total": subtotal + fee, "fulfillment_mode": "ready", "items": items,
		"created_at": created.UTC(), "updated_at": updated.UTC(), "seed_tag": seedTag,
		"checkout": bson.M{"scope": checkoutScope, "request_id": number, "request_hash": hashHex(number)},
	}
	if buyer != nil {
		doc["customer_id"] = buyer.ID
		recipient, tel, addr = buyer.Name, buyer.Phone, buyer.Address
	} else {
		recipient, tel = vietnameseName(r), phone(r)
		doc["guest_access_hash"] = hashHex(number + "/guest")
	}
	doc["shipping_address"] = bson.M{"recipient": recipient, "phone": tel, "line1": addr.Line1, "ward": addr.Ward, "city": addr.City}
	if r.Float64() < 0.18 {
		doc["note"] = notes[r.IntN(len(notes))]
	}
	if !paidAt.IsZero() {
		doc["paid_at"] = paidAt.UTC()
	}
	if !cancelledAt.IsZero() {
		doc["cancelled_at"] = cancelledAt.UTC()
		doc["inventory_restock"] = domainorder.InventoryRestocked
	}
	return doc
}

// lifecycle walks an order through the same transitions the admin API allows
// (pending → confirmed → shipping → delivered for COD, confirmed → delivered at
// the counter), with each step a plausible delay after the previous one and
// never later than now.
func lifecycle(r *rand.Rand, method string, created, now time.Time) (status, payment string, updated, paidAt, cancelledAt time.Time) {
	status, payment, updated = domainorder.StatusPending, domainorder.PaymentUnpaid, created
	step := func(minH, maxH float64) (time.Time, bool) {
		t := updated.Add(time.Duration((minH + r.Float64()*(maxH-minH)) * float64(time.Hour)))
		return t, t.Before(now)
	}
	if r.Float64() < 0.07 {
		if t, ok := step(0.5, 20); ok {
			return domainorder.StatusCancelled, payment, t, time.Time{}, t
		}
	}
	// Staff confirm orders during shop hours (8h–21h), so a late-evening order
	// waits as pending until the next morning.
	t, ok := step(0.3, 3)
	if h := t.In(vnZone).Hour(); h < 8 || h >= 21 {
		next := startOfDay(t)
		if h >= 21 {
			next = next.AddDate(0, 0, 1)
		}
		t = next.Add(8*time.Hour + time.Duration(r.IntN(90))*time.Minute)
		ok = t.Before(now)
	}
	if !ok {
		return
	}
	status, updated = domainorder.StatusConfirmed, t
	if method == domainorder.PaymentMethodCashAtCounter {
		if t, ok = step(2, 48); ok {
			status, payment, updated, paidAt = domainorder.StatusDelivered, domainorder.PaymentPaid, t, t
		}
		return
	}
	if t, ok = step(4, 30); !ok {
		return
	}
	status, updated = domainorder.StatusShipping, t
	if t, ok = step(24, 72); ok {
		status, payment, updated, paidAt = domainorder.StatusDelivered, domainorder.PaymentPaid, t, t
	}
	return
}

func pickVariant(r *rand.Rand, variants []variant) variant {
	weights := make([]float64, len(variants))
	for i, v := range variants {
		weights[i] = sizeWeight[v.Size]
		if weights[i] == 0 {
			weights[i] = 1
		}
	}
	return variants[weighted(r, weights)]
}

func weighted(r *rand.Rand, weights []float64) int {
	var total float64
	for _, w := range weights {
		total += w
	}
	x := r.Float64() * total
	for i, w := range weights {
		if x < w {
			return i
		}
		x -= w
	}
	return len(weights) - 1
}

func pickHour(r *rand.Rand) int { return weighted(r, hourWeight[:]) }

func startOfDay(t time.Time) time.Time {
	t = t.In(vnZone)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, vnZone)
}

func randomTime(r *rand.Rand, from, to time.Time) time.Time {
	return from.Add(time.Duration(r.Int64N(int64(to.Sub(from)))))
}

func vietnameseName(r *rand.Rand) string {
	family := familyNames[r.IntN(len(familyNames))]
	if r.Float64() < 0.88 {
		return family + " " + femaleMiddle[r.IntN(len(femaleMiddle))] + " " + femaleGiven[r.IntN(len(femaleGiven))]
	}
	return family + " " + maleMiddle[r.IntN(len(maleMiddle))] + " " + maleGiven[r.IntN(len(maleGiven))]
}

func phone(r *rand.Rand) string {
	prefixes := []string{"090", "091", "093", "094", "096", "097", "098", "032", "033", "035", "038", "070", "077", "079", "081", "083", "085"}
	return fmt.Sprintf("%s%07d", prefixes[r.IntN(len(prefixes))], r.IntN(10_000_000))
}

func randomAddress(r *rand.Rand, wards []ward) address {
	w := wards[r.IntN(len(wards))]
	return address{Line1: fmt.Sprintf("%d %s", 1+r.IntN(320), streets[r.IntN(len(streets))]), Ward: w.Name, City: w.Province}
}

func hashHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// asciiSlug turns "Nguyễn Thị Ánh" into "nguyen.thi.anh" for demo e-mails.
func asciiSlug(s, sep string) string {
	s = strings.NewReplacer("đ", "d", "Đ", "D").Replace(s)
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == ' ':
			b.WriteString(sep)
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
