package tailoring

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

const testID = "6d0000000000000000000001"

type fakeAdminStore struct {
	doc        *domain.TailoringRequest
	findErr    error
	updateErr  error
	failFirstN int // return ErrConcurrentUpdate for first N UpdateIf calls
	updates    int
}

func (f *fakeAdminStore) FindByID(_ context.Context, id string) (*domain.TailoringRequest, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.doc == nil || f.doc.ID != id {
		return nil, nil
	}
	c := *f.doc
	return &c, nil
}

func (f *fakeAdminStore) ListAdmin(context.Context, AdminFilter) ([]domain.TailoringRequest, int64, error) {
	return nil, 0, f.findErr
}

func (f *fakeAdminStore) UpdateIf(_ context.Context, cur, next *domain.TailoringRequest) error {
	f.updates++
	if f.updateErr != nil {
		return f.updateErr
	}
	if f.failFirstN > 0 {
		f.failFirstN--
		return ErrConcurrentUpdate
	}
	if f.doc.Status != cur.Status || !f.doc.UpdatedAt.Equal(cur.UpdatedAt) {
		return ErrConcurrentUpdate
	}
	c := *next
	f.doc = &c
	return nil
}

func fl(v float64) *float64 { return &v }

func fullMeasurements() *domain.Measurements {
	return &domain.Measurements{NeckCM: fl(34), BustCM: fl(86), WaistCM: fl(65), HipsCM: fl(92), ShoulderCM: fl(37), SleeveCM: fl(56), GarmentLengthCM: fl(138)}
}

func newAdmin(status string, m *domain.Measurements) (*AdminService, *fakeAdminStore) {
	st := &fakeAdminStore{doc: &domain.TailoringRequest{
		ID: testID, FullName: "A", Phone: "0900000001", Status: status, Measurements: m,
		CreatedAt: time.Unix(1000, 0).UTC(), UpdatedAt: time.Unix(1000, 0).UTC(),
	}}
	return NewAdminService(st), st
}

func TestAdminGetNotFound(t *testing.T) {
	svc, _ := newAdmin(domain.StatusSubmitted, nil)
	for _, id := range []string{"bad", "6d00000000000000000000ff"} {
		if _, err := svc.Get(context.Background(), id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("%s: expected ErrNotFound, got %v", id, err)
		}
	}
}

func TestAdminInfraErrorIsUnavailable(t *testing.T) {
	svc, st := newAdmin(domain.StatusSubmitted, nil)
	st.findErr = errors.New("conn refused")
	if _, err := svc.Get(context.Background(), testID); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("expected unavailable, got %v", err)
	}
	if _, _, _, _, err := svc.List(context.Background(), AdminFilter{}); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("list: expected unavailable, got %v", err)
	}
	st.findErr = nil
	st.updateErr = errors.New("write failed")
	if _, err := svc.SetStatus(context.Background(), testID, domain.StatusContacted); !errors.Is(err, domain.ErrTailoringUnavailable) {
		t.Fatalf("update: expected unavailable, got %v", err)
	}
}

func TestAdminListValidation(t *testing.T) {
	svc, _ := newAdmin(domain.StatusSubmitted, nil)
	var ife *domain.InvalidFieldError
	if _, _, _, _, err := svc.List(context.Background(), AdminFilter{Status: "bogus"}); !errors.As(err, &ife) || ife.Field != "status" {
		t.Fatalf("expected status invalid, got %v", err)
	}
	_, _, page, size, err := svc.List(context.Background(), AdminFilter{Status: "submitted", PageSize: 1000})
	if err != nil || page != 1 || size != 100 {
		t.Fatalf("defaults/clamp wrong: %v %d %d", err, page, size)
	}
}

func TestAdminSetStatusTransitions(t *testing.T) {
	ctx := context.Background()
	svc, st := newAdmin(domain.StatusSubmitted, nil)

	if _, err := svc.SetStatus(ctx, testID, domain.StatusConsultationScheduled); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("skip should conflict, got %v", err)
	}
	got, err := svc.SetStatus(ctx, testID, domain.StatusContacted)
	if err != nil || got.Status != domain.StatusContacted {
		t.Fatalf("submitted->contacted: %v %+v", err, got)
	}
	updates := st.updates
	if _, err := svc.SetStatus(ctx, testID, domain.StatusContacted); err != nil || st.updates != updates {
		t.Fatalf("same status must be a no-op without write: err=%v", err)
	}
	if _, err := svc.SetStatus(ctx, testID, domain.StatusSubmitted); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("backwards should conflict, got %v", err)
	}
	var ife *domain.InvalidFieldError
	if _, err := svc.SetStatus(ctx, testID, domain.StatusMeasurementConfirmed); !errors.As(err, &ife) {
		t.Fatalf("measurement_confirmed via status must be 400-type, got %v", err)
	}
	if _, err := svc.SetStatus(ctx, testID, "weird"); !errors.As(err, &ife) {
		t.Fatalf("unknown status must be invalid, got %v", err)
	}
	if got, err := svc.SetStatus(ctx, testID, domain.StatusCancelled); err != nil || got.Status != domain.StatusCancelled {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := svc.SetStatus(ctx, testID, domain.StatusContacted); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("cancelled is terminal, got %v", err)
	}
}

func TestAdminInProgressOnlyFromConfirmed(t *testing.T) {
	ctx := context.Background()
	svc, st := newAdmin(domain.StatusConsultationScheduled, fullMeasurements())
	if _, err := svc.SetStatus(ctx, testID, domain.StatusInProgress); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if _, err := svc.ConfirmMeasurements(ctx, testID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus(ctx, testID, domain.StatusInProgress); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus(ctx, testID, domain.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SetStatus(ctx, testID, domain.StatusCancelled); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("completed cannot cancel, got %v", err)
	}
	if st.doc.Status != domain.StatusCompleted {
		t.Fatal("status changed unexpectedly")
	}
}

func TestAdminConfirmMeasurements(t *testing.T) {
	ctx := context.Background()
	// incomplete
	m := fullMeasurements()
	m.SleeveCM = nil
	svc, st := newAdmin(domain.StatusContacted, m)
	if _, err := svc.ConfirmMeasurements(ctx, testID); !errors.Is(err, ErrMeasurementIncomplete) {
		t.Fatalf("expected incomplete, got %v", err)
	}
	if st.doc.Status != domain.StatusContacted {
		t.Fatal("must not change status")
	}
	// wrong state
	svc, _ = newAdmin(domain.StatusSubmitted, fullMeasurements())
	if _, err := svc.ConfirmMeasurements(ctx, testID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	// ok from contacted, idempotent retry
	svc, st = newAdmin(domain.StatusContacted, fullMeasurements())
	got, err := svc.ConfirmMeasurements(ctx, testID)
	if err != nil || got.Status != domain.StatusMeasurementConfirmed {
		t.Fatalf("confirm: %v", err)
	}
	n := st.updates
	if _, err := svc.ConfirmMeasurements(ctx, testID); err != nil || st.updates != n {
		t.Fatalf("retry must be no-op: %v", err)
	}
	// in_progress cannot confirm
	svc, _ = newAdmin(domain.StatusInProgress, fullMeasurements())
	if _, err := svc.ConfirmMeasurements(ctx, testID); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestAdminUpdateFields(t *testing.T) {
	ctx := context.Background()
	svc, st := newAdmin(domain.StatusSubmitted, &domain.Measurements{NeckCM: fl(34), WaistCM: fl(65)})
	note := "  ghi chú  "
	appt := time.Date(2026, 12, 1, 9, 0, 0, 0, time.FixedZone("x", 7*3600))
	staff := "Lan"
	zalo := "zalo"
	got, err := svc.Update(ctx, testID, UpdateInput{
		Notes: Optional[string]{Set: true, Value: &note}, AppointmentAt: Optional[time.Time]{Set: true, Value: &appt},
		AssignedStaff: Optional[string]{Set: true, Value: &staff}, PreferredContact: Optional[string]{Set: true, Value: &zalo},
		Measurements: &MeasurementsPatch{BustCM: Optional[float64]{Set: true, Value: fl(86.5)}, NeckCM: Optional[float64]{Set: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if *got.Notes != "ghi chú" || got.AppointmentAt.Location() != time.UTC || *got.AssignedStaff != "Lan" || *got.PreferredContact != "zalo" {
		t.Fatalf("fields wrong: %+v", got)
	}
	if got.Measurements.NeckCM != nil || *got.Measurements.BustCM != 86.5 || *got.Measurements.WaistCM != 65 {
		t.Fatalf("measurement merge wrong: %+v", got.Measurements)
	}
	if !got.UpdatedAt.After(time.Unix(1000, 0)) || st.doc.Status != domain.StatusSubmitted {
		t.Fatal("updated_at must advance, status unchanged")
	}
	// clear everything
	got, err = svc.Update(ctx, testID, UpdateInput{
		Notes: Optional[string]{Set: true}, AppointmentAt: Optional[time.Time]{Set: true}, AssignedStaff: Optional[string]{Set: true},
		Measurements: &MeasurementsPatch{BustCM: Optional[float64]{Set: true}, WaistCM: Optional[float64]{Set: true}},
	})
	if err != nil || got.Notes != nil || got.AppointmentAt != nil || got.AssignedStaff != nil || got.Measurements != nil {
		t.Fatalf("clear failed: %v %+v", err, got)
	}
}

func TestAdminUpdateValidationAndLock(t *testing.T) {
	ctx := context.Background()
	svc, _ := newAdmin(domain.StatusSubmitted, nil)
	var ife *domain.InvalidFieldError
	if _, err := svc.Update(ctx, testID, UpdateInput{}); !errors.As(err, &ife) {
		t.Fatalf("empty patch must be 400-type, got %v", err)
	}
	bad := "fax"
	if _, err := svc.Update(ctx, testID, UpdateInput{PreferredContact: Optional[string]{Set: true, Value: &bad}}); !errors.As(err, &ife) {
		t.Fatalf("bad contact: %v", err)
	}
	if _, err := svc.Update(ctx, testID, UpdateInput{Measurements: &MeasurementsPatch{WaistCM: Optional[float64]{Set: true, Value: fl(0)}}}); !errors.As(err, &ife) {
		t.Fatalf("zero measurement: %v", err)
	}
	for _, s := range []string{domain.StatusMeasurementConfirmed, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled} {
		svc, st := newAdmin(s, fullMeasurements())
		_, err := svc.Update(ctx, testID, UpdateInput{Measurements: &MeasurementsPatch{WaistCM: Optional[float64]{Set: true, Value: fl(70)}}})
		if !errors.Is(err, ErrMeasurementLocked) {
			t.Fatalf("%s: expected locked, got %v", s, err)
		}
		if *st.doc.Measurements.WaistCM != 65 {
			t.Fatalf("%s: measurements changed", s)
		}
		// non-measurement fields are still editable
		n := "ok"
		if _, err := svc.Update(ctx, testID, UpdateInput{Notes: Optional[string]{Set: true, Value: &n}}); err != nil {
			t.Fatalf("%s: notes edit should work: %v", s, err)
		}
	}
}

func TestAdminConcurrentRetryReevaluates(t *testing.T) {
	ctx := context.Background()
	svc, st := newAdmin(domain.StatusSubmitted, nil)
	st.failFirstN = 1
	if _, err := svc.SetStatus(ctx, testID, domain.StatusContacted); err != nil {
		t.Fatalf("should succeed on retry: %v", err)
	}
	if st.updates != 2 {
		t.Fatalf("expected 2 attempts, got %d", st.updates)
	}
	svc, st = newAdmin(domain.StatusSubmitted, nil)
	st.failFirstN = 10
	if _, err := svc.SetStatus(ctx, testID, domain.StatusContacted); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("persistent contention should conflict, got %v", err)
	}
}
