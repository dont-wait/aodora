package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"
)

func vn(y int, m time.Month, d, h int) time.Time { return time.Date(y, m, d, h, 0, 0, 0, Location()) }

func TestResolveWindowDefaultsToLast30DaysVietnamTime(t *testing.T) {
	// 2026-11-27 18:30 UTC is already 2026-11-28 01:30 in Vietnam.
	now := time.Date(2026, 11, 27, 18, 30, 0, 0, time.UTC)
	w, err := ResolveWindow(now, "", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !w.End.Equal(vn(2026, 11, 29, 0)) || !w.Start.Equal(vn(2026, 10, 30, 0)) {
		t.Fatalf("window = %v .. %v", w.Start, w.End)
	}
	if !w.PrevEnd.Equal(w.Start) || !w.PrevStart.Equal(vn(2026, 9, 30, 0)) {
		t.Fatalf("previous = %v .. %v", w.PrevStart, w.PrevEnd)
	}
	if w.Granularity != Day || w.LowStock != DefaultLowStock {
		t.Fatalf("defaults: %v %v", w.Granularity, w.LowStock)
	}
}

func TestResolveWindowValidation(t *testing.T) {
	now := time.Date(2026, 11, 27, 5, 0, 0, 0, time.UTC)
	for name, args := range map[string][4]string{
		"bad from":    {"2026/01/01", "2026-02-01", "day", ""},
		"bad to":      {"2026-01-01", "nope", "day", ""},
		"from > to":   {"2026-02-02", "2026-02-01", "day", ""},
		"too long":    {"2025-01-01", "2026-06-01", "day", ""},
		"granularity": {"2026-01-01", "2026-02-01", "hour", ""},
	} {
		if _, err := ResolveWindow(now, args[0], args[1], args[2], 0); err == nil {
			t.Errorf("%s: want error", name)
		} else {
			var invalid Invalid
			if !errors.As(err, &invalid) {
				t.Errorf("%s: want Invalid, got %v", name, err)
			}
		}
	}
	if _, err := ResolveWindow(now, "2025-11-27", "2026-11-27", "month", 0); err != nil {
		t.Fatalf("366 days must be accepted: %v", err)
	}
	if _, err := ResolveWindow(now, "2025-11-26", "2026-11-27", "day", 0); err == nil {
		t.Fatal("367 days must be rejected")
	}
	if _, err := ResolveWindow(now, "2026-01-01", "2026-01-31", "day", -1); err == nil {
		t.Fatal("negative threshold must be rejected")
	}
}

func TestPreviousPeriodHasSameLength(t *testing.T) {
	w, err := ResolveWindow(time.Now(), "2026-03-01", "2026-03-10", "day", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := w.End.Sub(w.Start); got != w.PrevEnd.Sub(w.PrevStart) {
		t.Fatalf("lengths differ: %v vs %v", got, w.PrevEnd.Sub(w.PrevStart))
	}
	if !w.PrevStart.Equal(vn(2026, 2, 19, 0)) {
		t.Fatalf("prev start = %v", w.PrevStart)
	}
}

func TestFillSeriesZeroFillsEmptyBuckets(t *testing.T) {
	start, end := vn(2026, 3, 1, 0), vn(2026, 3, 6, 0)
	got := FillSeries(start, end, Day, []Bucket{{Start: vn(2026, 3, 2, 0), Orders: 2, Revenue: 500}, {Start: vn(2026, 3, 5, 0), Orders: 1, Revenue: 100}})
	if len(got) != 5 {
		t.Fatalf("len = %d: %+v", len(got), got)
	}
	want := []Point{{"2026-03-01", 0, 0}, {"2026-03-02", 500, 2}, {"2026-03-03", 0, 0}, {"2026-03-04", 0, 0}, {"2026-03-05", 100, 1}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("point %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestFillSeriesWeeksStartOnMondayAndMonthsOnFirst(t *testing.T) {
	// 2026-03-04 is a Wednesday; its week starts Monday 2026-03-02.
	weeks := FillSeries(vn(2026, 3, 4, 0), vn(2026, 3, 20, 0), Week, nil)
	if len(weeks) != 3 || weeks[0].Date != "2026-03-02" || weeks[1].Date != "2026-03-09" || weeks[2].Date != "2026-03-16" {
		t.Fatalf("weeks = %+v", weeks)
	}
	months := FillSeries(vn(2026, 1, 15, 0), vn(2026, 4, 2, 0), Month, []Bucket{{Start: vn(2026, 2, 1, 0), Orders: 3, Revenue: 9}})
	if len(months) != 4 || months[0].Date != "2026-01-01" || months[1].Orders != 3 || months[3].Date != "2026-04-01" {
		t.Fatalf("months = %+v", months)
	}
}

func TestChangePct(t *testing.T) {
	if got := ChangePct(150, 100); got == nil || *got != 50 {
		t.Fatalf("up: %v", got)
	}
	if got := ChangePct(50, 100); got == nil || *got != -50 {
		t.Fatalf("down: %v", got)
	}
	if ChangePct(10, 0) != nil {
		t.Fatal("no baseline must give nil")
	}
	if got := ChangePct(0, 0); got != nil {
		t.Fatalf("0/0: %v", got)
	}
}

type fakeRepo struct {
	raw Raw
	err error
	got Window
}

func (f *fakeRepo) Aggregate(_ context.Context, w Window) (Raw, error) {
	f.got = w
	return f.raw, f.err
}

func TestReportDefinesKPIsFromRawPeriods(t *testing.T) {
	repo := &fakeRepo{raw: Raw{
		Current:  Period{Orders: 10, Cancelled: 2, PaidOrders: 4, Revenue: 4_000_000, NewCustomers: 6},
		Previous: Period{Orders: 8, Cancelled: 4, PaidOrders: 5, Revenue: 2_500_000, NewCustomers: 0},
	}}
	s := NewService(repo)
	s.now = func() time.Time { return time.Date(2026, 3, 10, 5, 0, 0, 0, time.UTC) }
	report, err := s.Report(context.Background(), "2026-03-01", "2026-03-10", "day", 0)
	if err != nil {
		t.Fatal(err)
	}
	k := report.KPIs
	if k.Revenue.Value != 4_000_000 || k.Revenue.ChangePct == nil || *k.Revenue.ChangePct != 60 {
		t.Fatalf("revenue = %+v", k.Revenue)
	}
	// Average order value = revenue / paid (revenue-counted) orders: 1,000,000 vs 500,000.
	if k.AvgOrderValue.Value != 1_000_000 || k.AvgOrderValue.Previous != 500_000 || *k.AvgOrderValue.ChangePct != 100 {
		t.Fatalf("aov = %+v", k.AvgOrderValue)
	}
	if k.CancelRate.Value != 0.2 || k.CancelRate.Previous != 0.5 {
		t.Fatalf("cancel = %+v", k.CancelRate)
	}
	if k.NewCustomers.Value != 6 || k.NewCustomers.ChangePct != nil {
		t.Fatalf("new customers must have no change when previous is 0: %+v", k.NewCustomers)
	}
	if report.Range.From != "2026-03-01" || report.Range.To != "2026-03-10" || report.Range.PrevFrom != "2026-02-19" || report.Range.PrevTo != "2026-02-28" {
		t.Fatalf("range = %+v", report.Range)
	}
	if len(report.Series) != 10 || report.OrdersByStatus == nil || report.Inventory.LowStock == nil || report.TopRevenue == nil || report.BySize == nil {
		t.Fatalf("empty collections must be [] not null: %+v", report)
	}
}

func TestReportZeroPeriodsDoNotDivideByZero(t *testing.T) {
	report := Shape(Window{Start: vn(2026, 3, 1, 0), End: vn(2026, 3, 2, 0), PrevStart: vn(2026, 2, 28, 0), PrevEnd: vn(2026, 3, 1, 0), Granularity: Day}, Raw{})
	if report.KPIs.AvgOrderValue.Value != 0 || report.KPIs.CancelRate.Value != 0 || report.KPIs.Revenue.ChangePct != nil {
		t.Fatalf("kpis = %+v", report.KPIs)
	}
}

func TestReportPropagatesRepositoryError(t *testing.T) {
	s := NewService(&fakeRepo{err: ErrUnavailable})
	if _, err := s.Report(context.Background(), "", "", "", 0); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.Report(context.Background(), "x", "", "", 0); err == nil {
		t.Fatal("invalid query must fail before hitting the repository")
	}
}
