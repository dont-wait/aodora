// Package dashboard builds the owner's statistics report. Revenue definition
// (see docs/ADMIN_DASHBOARD_IMPORT.md): the sum of orders.total of orders whose
// payment_status is "paid" and whose status is not "cancelled".
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Granularity string

const (
	Day   Granularity = "day"
	Week  Granularity = "week"
	Month Granularity = "month"

	// MaxRangeDays bounds one report so the pipelines stay cheap.
	MaxRangeDays      = 366
	DefaultRangeDays  = 30
	DefaultLowStock   = 3
	Timezone          = "Asia/Ho_Chi_Minh"
	timezoneOffsetSec = 7 * 3600
)

// Invalid is returned for bad query parameters.
type Invalid struct{ Field, Message string }

func (e Invalid) Error() string { return e.Field + ": " + e.Message }

var ErrUnavailable = errors.New("dashboard unavailable")

// Location returns Vietnam time. Vietnam has no DST, so a fixed +07:00 zone is exact and needs no tzdata.
func Location() *time.Location { return time.FixedZone("ICT", timezoneOffsetSec) }

// Window is the resolved period: [Start, End) plus the previous period of equal length.
type Window struct {
	Start, End         time.Time
	PrevStart, PrevEnd time.Time
	Granularity        Granularity
	LowStock           int64
}

// Count is a keyed counter (status, payment method, ...).
type Count struct {
	Key     string `json:"key"`
	Count   int64  `json:"count"`
	Revenue int64  `json:"revenue"`
}

// Period holds the KPI inputs of one period.
type Period struct {
	Orders       int64 `json:"orders"`
	Cancelled    int64 `json:"cancelled"`
	PaidOrders   int64 `json:"paid_orders"`
	Revenue      int64 `json:"revenue"`
	NewCustomers int64 `json:"new_customers"`
}

// Bucket is one raw time-series bucket keyed by its start instant.
type Bucket struct {
	Start   time.Time
	Orders  int64
	Revenue int64
}

type ProductStat struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug,omitempty"`
	Image     string `json:"image,omitempty"`
	Revenue   int64  `json:"revenue"`
	Quantity  int64  `json:"quantity"`
}
type CategoryStat struct {
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	Revenue    int64  `json:"revenue"`
	Quantity   int64  `json:"quantity"`
}
type SizeStat struct {
	Size     string `json:"size"`
	Revenue  int64  `json:"revenue"`
	Quantity int64  `json:"quantity"`
}
type LowStockItem struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SKU       string `json:"sku"`
	Color     string `json:"color"`
	Size      string `json:"size"`
	Stock     int64  `json:"stock"`
}
type Inventory struct {
	LowStockThreshold int64          `json:"low_stock_threshold"`
	ActiveProducts    int64          `json:"active_products"`
	HiddenProducts    int64          `json:"hidden_products"`
	StockValue        int64          `json:"stock_value"`
	OutOfStockSKUs    int64          `json:"out_of_stock_skus"`
	LowStockSKUs      int64          `json:"low_stock_skus"`
	LowStock          []LowStockItem `json:"low_stock"`
}

// Raw is what a repository returns; the service shapes it into a Report.
type Raw struct {
	Current, Previous Period
	Series            []Bucket
	ByStatus          []Count
	ByPaymentStatus   []Count
	ByPaymentMethod   []Count
	TopRevenue        []ProductStat
	TopQuantity       []ProductStat
	ByCategory        []CategoryStat
	BySize            []SizeStat
	Inventory         Inventory
	Tailoring         []Count
}

type Repository interface {
	Aggregate(ctx context.Context, w Window) (Raw, error)
}

type Metric struct {
	Value     float64  `json:"value"`
	Previous  float64  `json:"previous"`
	ChangePct *float64 `json:"change_pct"`
}
type KPIs struct {
	Revenue       Metric `json:"revenue"`
	Orders        Metric `json:"orders"`
	AvgOrderValue Metric `json:"avg_order_value"`
	NewCustomers  Metric `json:"new_customers"`
	CancelRate    Metric `json:"cancel_rate"`
}
type Point struct {
	Date    string `json:"date"`
	Revenue int64  `json:"revenue"`
	Orders  int64  `json:"orders"`
}
type Range struct {
	From        string      `json:"from"`
	To          string      `json:"to"`
	PrevFrom    string      `json:"prev_from"`
	PrevTo      string      `json:"prev_to"`
	Granularity Granularity `json:"granularity"`
	Timezone    string      `json:"timezone"`
}
type Report struct {
	Range          Range          `json:"range"`
	KPIs           KPIs           `json:"kpis"`
	Series         []Point        `json:"series"`
	OrdersByStatus []Count        `json:"orders_by_status"`
	OrdersByPay    []Count        `json:"orders_by_payment_status"`
	OrdersByMethod []Count        `json:"orders_by_payment_method"`
	TopRevenue     []ProductStat  `json:"top_products_revenue"`
	TopQuantity    []ProductStat  `json:"top_products_quantity"`
	ByCategory     []CategoryStat `json:"revenue_by_category"`
	BySize         []SizeStat     `json:"revenue_by_size"`
	Inventory      Inventory      `json:"inventory"`
	Tailoring      []Count        `json:"tailoring_by_status"`
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

const dateLayout = "2006-01-02"

// ResolveWindow validates the query. from/to are inclusive local dates; empty
// values default to the last 30 days ending today (Vietnam time).
func ResolveWindow(now time.Time, from, to, granularity string, lowStock int64) (Window, error) {
	loc := Location()
	today := dayStart(now.In(loc))
	end := today
	if to != "" {
		t, err := time.ParseInLocation(dateLayout, to, loc)
		if err != nil {
			return Window{}, Invalid{"to", "Ngày phải có dạng YYYY-MM-DD"}
		}
		end = t
	}
	start := end.AddDate(0, 0, -(DefaultRangeDays - 1))
	if from != "" {
		t, err := time.ParseInLocation(dateLayout, from, loc)
		if err != nil {
			return Window{}, Invalid{"from", "Ngày phải có dạng YYYY-MM-DD"}
		}
		start = t
	}
	if start.After(end) {
		return Window{}, Invalid{"from", "Ngày bắt đầu phải trước hoặc bằng ngày kết thúc"}
	}
	endExclusive := end.AddDate(0, 0, 1)
	days := int(endExclusive.Sub(start).Hours()/24 + 0.5)
	if days > MaxRangeDays {
		return Window{}, Invalid{"from", fmt.Sprintf("Khoảng thời gian tối đa %d ngày", MaxRangeDays)}
	}
	g := Granularity(granularity)
	switch g {
	case "":
		g = Day
	case Day, Week, Month:
	default:
		return Window{}, Invalid{"granularity", "Phải là day, week hoặc month"}
	}
	if lowStock == 0 {
		lowStock = DefaultLowStock
	}
	if lowStock < 0 || lowStock > 1000 {
		return Window{}, Invalid{"low_stock", "Ngưỡng tồn phải từ 0 đến 1000"}
	}
	return Window{
		Start: start, End: endExclusive,
		PrevStart: start.AddDate(0, 0, -days), PrevEnd: start,
		Granularity: g, LowStock: lowStock,
	}, nil
}

func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// BucketStart returns the start of the bucket containing t (weeks start on Monday).
func BucketStart(t time.Time, g Granularity) time.Time {
	t = t.In(Location())
	switch g {
	case Week:
		d := dayStart(t)
		offset := (int(d.Weekday()) + 6) % 7
		return d.AddDate(0, 0, -offset)
	case Month:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	default:
		return dayStart(t)
	}
}

func nextBucket(t time.Time, g Granularity) time.Time {
	switch g {
	case Week:
		return t.AddDate(0, 0, 7)
	case Month:
		return t.AddDate(0, 1, 0)
	default:
		return t.AddDate(0, 0, 1)
	}
}

// FillSeries returns every bucket between start and end (exclusive), zero-filled.
func FillSeries(start, end time.Time, g Granularity, raw []Bucket) []Point {
	byStart := make(map[int64]Bucket, len(raw))
	for _, b := range raw {
		byStart[BucketStart(b.Start, g).Unix()] = b
	}
	out := []Point{}
	for t := BucketStart(start, g); t.Before(end); t = nextBucket(t, g) {
		b := byStart[t.Unix()]
		out = append(out, Point{Date: t.Format(dateLayout), Revenue: b.Revenue, Orders: b.Orders})
	}
	return out
}

// ChangePct is (current-previous)/previous*100, or nil when there is no baseline.
func ChangePct(current, previous float64) *float64 {
	if previous == 0 {
		return nil
	}
	v := (current - previous) / previous * 100
	return &v
}

func metric(current, previous float64) Metric {
	return Metric{Value: current, Previous: previous, ChangePct: ChangePct(current, previous)}
}

func avg(revenue, orders int64) float64 {
	if orders == 0 {
		return 0
	}
	return float64(revenue) / float64(orders)
}

func rate(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole)
}

// Shape turns raw aggregates into the response (pure, unit tested).
func Shape(w Window, raw Raw) Report {
	cur, prev := raw.Current, raw.Previous
	nonNil := func(c []Count) []Count {
		if c == nil {
			return []Count{}
		}
		return c
	}
	report := Report{
		Range: Range{
			From: w.Start.Format(dateLayout), To: w.End.AddDate(0, 0, -1).Format(dateLayout),
			PrevFrom: w.PrevStart.Format(dateLayout), PrevTo: w.PrevEnd.AddDate(0, 0, -1).Format(dateLayout),
			Granularity: w.Granularity, Timezone: Timezone,
		},
		KPIs: KPIs{
			Revenue:       metric(float64(cur.Revenue), float64(prev.Revenue)),
			Orders:        metric(float64(cur.Orders), float64(prev.Orders)),
			AvgOrderValue: metric(avg(cur.Revenue, cur.PaidOrders), avg(prev.Revenue, prev.PaidOrders)),
			NewCustomers:  metric(float64(cur.NewCustomers), float64(prev.NewCustomers)),
			CancelRate:    metric(rate(cur.Cancelled, cur.Orders), rate(prev.Cancelled, prev.Orders)),
		},
		Series:         FillSeries(w.Start, w.End, w.Granularity, raw.Series),
		OrdersByStatus: nonNil(raw.ByStatus),
		OrdersByPay:    nonNil(raw.ByPaymentStatus),
		OrdersByMethod: nonNil(raw.ByPaymentMethod),
		TopRevenue:     raw.TopRevenue,
		TopQuantity:    raw.TopQuantity,
		ByCategory:     raw.ByCategory,
		BySize:         raw.BySize,
		Inventory:      raw.Inventory,
		Tailoring:      nonNil(raw.Tailoring),
	}
	if report.TopRevenue == nil {
		report.TopRevenue = []ProductStat{}
	}
	if report.TopQuantity == nil {
		report.TopQuantity = []ProductStat{}
	}
	if report.ByCategory == nil {
		report.ByCategory = []CategoryStat{}
	}
	if report.BySize == nil {
		report.BySize = []SizeStat{}
	}
	if report.Inventory.LowStock == nil {
		report.Inventory.LowStock = []LowStockItem{}
	}
	return report
}

func (s *Service) Report(ctx context.Context, from, to, granularity string, lowStock int64) (Report, error) {
	w, err := ResolveWindow(s.now(), from, to, granularity, lowStock)
	if err != nil {
		return Report{}, err
	}
	raw, err := s.repo.Aggregate(ctx, w)
	if err != nil {
		return Report{}, err
	}
	return Shape(w, raw), nil
}
