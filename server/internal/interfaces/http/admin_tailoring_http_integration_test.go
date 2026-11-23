//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	auth "github.com/your-org/aodora/server/internal/application/auth"
	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	mongorepo "github.com/your-org/aodora/server/internal/infrastructure/mongo"
	"github.com/your-org/aodora/server/internal/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAdminTailoringHTTPEndToEndIntegration(t *testing.T) {
	_, db := httpIntegrationTestDatabase(t)
	ctx := context.Background()
	for _, col := range []string{"tailoring_requests", "customers", "sessions"} {
		if _, err := db.Collection(col).DeleteMany(ctx, bson.M{}); err != nil {
			t.Fatal(err)
		}
	}
	repo := mongorepo.NewTailoringRepository(db)
	authRepo := mongorepo.NewAuthRepository(db)
	guard := NewAuthGuard(auth.NewService(authRepo, time.Hour), AuthCookieConfig{Name: "session", TTL: time.Hour, AllowedOrigin: "https://shop.example"})
	log := *logger.NewDiscard()
	router := NewRouter(nil, log, WithAuth(guard), WithTailoring(apptailoring.NewService(repo), guard, log),
		WithAdminCatalog(nil, guard.RequireAdmin, log, "https://shop.example"),
		WithAdminTailoring(apptailoring.NewAdminService(repo), guard.RequireAdmin, log, "https://shop.example"))

	do := func(method, path, body string, cookie *http.Cookie, status int) []byte {
		t.Helper()
		w := adminOrderRequest(router, method, path, body, cookie, nil)
		if w.Code != status {
			t.Fatalf("%s %s: status %d want %d response=%s", method, path, w.Code, status, w.Body.String())
		}
		if strings.HasPrefix(path, "/api/v1/admin/") && w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s: missing no-store", method, path)
		}
		return w.Body.Bytes()
	}
	errCode := func(raw []byte) string {
		var e struct {
			Error apiError `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return e.Error.Code
	}

	for _, email := range []string{"admin-tailoring@example.com", "customer-tailoring@example.com"} {
		do("POST", "/api/v1/auth/register", fmt.Sprintf(`{"full_name":"T Test","email":%q,"password":"password-123456"}`, email), nil, 201)
	}
	if res, err := db.Collection("customers").UpdateOne(ctx, bson.M{"email": "admin-tailoring@example.com"}, bson.M{"$set": bson.M{"role": "admin"}}); err != nil || res.MatchedCount != 1 {
		t.Fatalf("seed admin role: %v", err)
	}
	login := func(email string) *http.Cookie {
		w := adminOrderRequest(router, "POST", "/api/v1/auth/login", fmt.Sprintf(`{"email":%q,"password":"password-123456"}`, email), nil, nil)
		if w.Code != 200 || len(w.Result().Cookies()) != 1 {
			t.Fatalf("login failed: %d", w.Code)
		}
		return w.Result().Cookies()[0]
	}
	admin := login("admin-tailoring@example.com")
	customer := login("customer-tailoring@example.com")

	base := "/api/v1/admin/tailoring-requests"
	// 403 for guest and ordinary customer on every route.
	for _, c := range []*http.Cookie{nil, customer} {
		if errCode(do("GET", base, "", c, 403)) != "ADMIN_REQUIRED" {
			t.Fatal("expected ADMIN_REQUIRED")
		}
		do("PATCH", base+"/6d0000000000000000000001", `{"notes":"x"}`, c, 403)
		do("PATCH", base+"/6d0000000000000000000001/status", `{"status":"contacted"}`, c, 403)
		do("POST", base+"/6d0000000000000000000001/confirm-measurements", `{"confirmed":true}`, c, 403)
	}

	// Seed three requests with identical created_at to prove stable pagination,
	// plus one older. Public API creates the first (guest, with product-less body).
	var created struct {
		Data struct{ ID string } `json:"data"`
	}
	raw := do("POST", "/api/v1/tailoring-requests", `{"full_name":"Khách Một","phone":"0912345678","notes":"muốn áo dài","measurements":{"body":"1m60","waist_cm":65.5}}`, nil, 201)
	_ = json.Unmarshal(raw, &created)
	id := created.Data.ID
	same := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	var ids []string
	for i := 0; i < 3; i++ {
		oid := bson.NewObjectID()
		ids = append(ids, oid.Hex())
		if _, err := db.Collection("tailoring_requests").InsertOne(ctx, bson.M{"_id": oid, "full_name": fmt.Sprintf("Seed %d", i), "phone": "0900000000", "status": "contacted", "created_at": same, "updated_at": same}); err != nil {
			t.Fatal(err)
		}
	}

	type listResp struct {
		Data  []adminTailoringDTO `json:"data"`
		Total int64               `json:"total"`
		Page  int64               `json:"page"`
		Size  int64               `json:"page_size"`
	}
	list := func(q string) listResp {
		var out listResp
		if err := json.Unmarshal(do("GET", base+q, "", admin, 200), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	if l := list(""); l.Total != 4 || len(l.Data) != 4 || l.Page != 1 || l.Size != 20 {
		t.Fatalf("list all: %+v", l)
	}
	if l := list("?status=submitted"); l.Total != 1 || l.Data[0].ID != id {
		t.Fatalf("filter submitted: %+v", l)
	}
	if l := list("?status=contacted"); l.Total != 3 {
		t.Fatalf("filter contacted: %+v", l)
	}
	if l := list("?status=cancelled"); l.Total != 0 || len(l.Data) != 0 {
		t.Fatalf("filter cancelled: %+v", l)
	}
	if errCode(do("GET", base+"?status=bogus", "", admin, 400)) != "INVALID_FIELD" {
		t.Fatal("bogus status must be 400")
	}
	// Same created_at: pages of 1 must cover all three seeds exactly once, _id desc.
	seen := map[string]bool{}
	prev := "ffffffffffffffffffffffff"
	for p := 1; p <= 3; p++ {
		l := list(fmt.Sprintf("?status=contacted&page=%d&page_size=1", p))
		if len(l.Data) != 1 || l.Total != 3 {
			t.Fatalf("page %d: %+v", p, l)
		}
		got := l.Data[0].ID
		if seen[got] || got >= prev {
			t.Fatalf("unstable order at page %d: %s", p, got)
		}
		seen[got], prev = true, got
	}

	// Detail + 404.
	// Always decode into a fresh value: omitted fields would otherwise keep stale data.
	var detail struct{ Data adminTailoringDTO }
	reload := func(id string) adminTailoringDTO {
		detail = struct{ Data adminTailoringDTO }{}
		if err := json.Unmarshal(do("GET", base+"/"+id, "", admin, 200), &detail); err != nil {
			t.Fatal(err)
		}
		return detail.Data
	}
	d := reload(id)
	if d.FullName != "Khách Một" || d.Phone != "0912345678" || !d.IsGuest || d.Status != "submitted" || *d.Notes != "muốn áo dài" || *d.Measurements.WaistCM != 65.5 || *d.Measurements.Body != "1m60" {
		t.Fatalf("detail: %+v", d)
	}
	do("GET", base+"/6d00000000000000000000ff", "", admin, 404)
	do("GET", base+"/not-an-id", "", admin, 404)

	// Confirm before measurements complete / wrong state.
	do("POST", base+"/"+id+"/confirm-measurements", `{"confirmed":true}`, admin, 409) // submitted
	do("PATCH", base+"/"+id+"/status", `{"status":"consultation_scheduled"}`, admin, 409)
	do("PATCH", base+"/"+id+"/status", `{"status":"measurement_confirmed"}`, admin, 400)
	do("PATCH", base+"/"+id+"/status", `{"status":"contacted"}`, admin, 200)
	do("PATCH", base+"/"+id+"/status", `{"status":"contacted"}`, admin, 200) // no-op
	if code := errCode(do("POST", base+"/"+id+"/confirm-measurements", `{"confirmed":true}`, admin, 409)); code != "MEASUREMENT_INCOMPLETE" {
		t.Fatalf("expected MEASUREMENT_INCOMPLETE, got %s", code)
	}

	// Patch each field, then clear some, reload to prove persistence.
	do("PATCH", base+"/"+id, `{"notes":"đã gọi","appointment_at":"2026-12-01T09:00:00+07:00","assigned_staff":"Lan","preferred_contact":"zalo","measurements":{"neck_cm":34,"bust_cm":86,"hips_cm":92,"shoulder_cm":37,"sleeve_cm":56,"garment_length_cm":138}}`, admin, 200)
	d = reload(id)
	if *d.Notes != "đã gọi" || d.AppointmentAt == nil || !d.AppointmentAt.Equal(time.Date(2026, 12, 1, 2, 0, 0, 0, time.UTC)) || *d.AssignedStaff != "Lan" || *d.PreferredContact != "zalo" || *d.Measurements.WaistCM != 65.5 || *d.Measurements.BustCM != 86 {
		t.Fatalf("after patch: %+v", d)
	}
	do("PATCH", base+"/"+id, `{"appointment_at":null,"assigned_staff":null,"preferred_contact":null,"measurements":{"body":null,"neck_cm":null}}`, admin, 200)
	d = reload(id)
	if d.AppointmentAt != nil || d.AssignedStaff != nil || d.PreferredContact != nil || d.Measurements.NeckCM != nil || d.Measurements.Body != nil || *d.Measurements.WaistCM != 65.5 {
		t.Fatalf("after clear: m=%+v appt=%v staff=%v pc=%v", *d.Measurements, d.AppointmentAt, d.AssignedStaff, d.PreferredContact)
	}
	// Invalid patches never write.
	do("PATCH", base+"/"+id, `{"measurements":{"waist_cm":0}}`, admin, 400)
	do("PATCH", base+"/"+id, `{"measurements":{"waist_cm":"70"}}`, admin, 400)
	do("PATCH", base+"/"+id, `{"status":"completed"}`, admin, 400)
	do("PATCH", base+"/"+id, `{"preferred_contact":"fax"}`, admin, 400)
	reload(id)
	if *detail.Data.Measurements.WaistCM != 65.5 || detail.Data.Status != "contacted" {
		t.Fatal("invalid patch changed data")
	}

	// Complete measurements, schedule, confirm, then locked.
	do("PATCH", base+"/"+id, `{"measurements":{"neck_cm":34}}`, admin, 200)
	do("PATCH", base+"/"+id+"/status", `{"status":"consultation_scheduled"}`, admin, 200)
	var confirmed struct{ Data adminTailoringDTO }
	_ = json.Unmarshal(do("POST", base+"/"+id+"/confirm-measurements", `{"confirmed":true}`, admin, 200), &confirmed)
	if confirmed.Data.Status != "measurement_confirmed" {
		t.Fatalf("confirm: %+v", confirmed.Data)
	}
	do("POST", base+"/"+id+"/confirm-measurements", `{"confirmed":true}`, admin, 200) // idempotent
	if code := errCode(do("PATCH", base+"/"+id, `{"measurements":{"waist_cm":70}}`, admin, 409)); code != "MEASUREMENT_LOCKED" {
		t.Fatalf("expected MEASUREMENT_LOCKED, got %s", code)
	}
	do("PATCH", base+"/"+id, `{"notes":"vẫn sửa được ghi chú"}`, admin, 200)
	do("PATCH", base+"/"+id+"/status", `{"status":"completed"}`, admin, 409)
	do("PATCH", base+"/"+id+"/status", `{"status":"in_progress"}`, admin, 200)
	do("PATCH", base+"/"+id+"/status", `{"status":"completed"}`, admin, 200)
	do("PATCH", base+"/"+id+"/status", `{"status":"cancelled"}`, admin, 409)
	do("PATCH", base+"/"+id+"/status", `{"status":"in_progress"}`, admin, 409)
	reload(id)
	if detail.Data.Status != "completed" || *detail.Data.Notes != "vẫn sửa được ghi chú" || *detail.Data.Measurements.WaistCM != 65.5 {
		t.Fatalf("final: %+v", detail.Data)
	}

	// Concurrent writers: exactly one wins a conflicting transition; no lost update.
	cid := ids[0]
	results := make(chan int, 2)
	for _, target := range []string{"consultation_scheduled", "cancelled"} {
		go func(target string) {
			w := adminOrderRequest(router, "PATCH", base+"/"+cid+"/status", fmt.Sprintf(`{"status":%q}`, target), admin, nil)
			results <- w.Code
		}(target)
	}
	a, b := <-results, <-results
	if a != 200 && b != 200 {
		t.Fatalf("one concurrent transition must succeed: %d %d", a, b)
	}
	if st := reload(cid).Status; st != "consultation_scheduled" && st != "cancelled" {
		t.Fatalf("final status %s", st)
	}
	// Concurrent patches of different fields must both persist.
	pid := ids[1]
	done := make(chan int, 2)
	for _, body := range []string{`{"notes":"ghi chú A"}`, `{"assigned_staff":"Mai"}`} {
		go func(body string) { done <- adminOrderRequest(router, "PATCH", base+"/"+pid, body, admin, nil).Code }(body)
	}
	if c1, c2 := <-done, <-done; c1 != 200 || c2 != 200 {
		t.Fatalf("concurrent patches: %d %d", c1, c2)
	}
	if f := reload(pid); f.Notes == nil || *f.Notes != "ghi chú A" || f.AssignedStaff == nil || *f.AssignedStaff != "Mai" {
		t.Fatalf("lost update: %+v", f)
	}
}
