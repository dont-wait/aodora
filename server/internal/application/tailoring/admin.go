package tailoring

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	domain "github.com/your-org/aodora/server/internal/domain/tailoring"
)

var (
	ErrNotFound              = errors.New("tailoring request not found")
	ErrStateConflict         = errors.New("tailoring state conflict")
	ErrMeasurementLocked     = errors.New("tailoring measurements locked")
	ErrMeasurementIncomplete = errors.New("tailoring measurements incomplete")
	// ErrConcurrentUpdate is returned by the store when the conditional write
	// matched nothing because the document changed (or vanished) since it was read.
	ErrConcurrentUpdate = errors.New("tailoring concurrent update")
)

const (
	MaxAssignedStaffLen = 200
	maxWriteAttempts    = 3
)

// Optional carries a PATCH field: Set=false means unchanged, Set=true with
// nil Value means clear.
type Optional[T any] struct {
	Set   bool
	Value *T
}

// MeasurementsPatch is a partial measurements update.
type MeasurementsPatch struct {
	Body            Optional[string]
	NeckCM          Optional[float64]
	BustCM          Optional[float64]
	WaistCM         Optional[float64]
	HipsCM          Optional[float64]
	ShoulderCM      Optional[float64]
	SleeveCM        Optional[float64]
	GarmentLengthCM Optional[float64]
}

func (p *MeasurementsPatch) any() bool {
	return p.Body.Set || p.NeckCM.Set || p.BustCM.Set || p.WaistCM.Set || p.HipsCM.Set ||
		p.ShoulderCM.Set || p.SleeveCM.Set || p.GarmentLengthCM.Set
}

// UpdateInput is a partial admin update. Status is deliberately absent.
type UpdateInput struct {
	Notes            Optional[string]
	AppointmentAt    Optional[time.Time]
	AssignedStaff    Optional[string]
	PreferredContact Optional[string]
	Measurements     *MeasurementsPatch
}

func (u UpdateInput) empty() bool {
	return !u.Notes.Set && !u.AppointmentAt.Set && !u.AssignedStaff.Set && !u.PreferredContact.Set &&
		(u.Measurements == nil || !u.Measurements.any())
}

type AdminFilter struct {
	Status   string
	Page     int64
	PageSize int64
}

// AdminStore owns reads and conditional writes.
type AdminStore interface {
	FindByID(ctx context.Context, id string) (*domain.TailoringRequest, error)
	ListAdmin(ctx context.Context, f AdminFilter) ([]domain.TailoringRequest, int64, error)
	// UpdateIf persists next only if the stored document still has
	// cur.Status and cur.UpdatedAt; otherwise it returns ErrConcurrentUpdate.
	UpdateIf(ctx context.Context, cur, next *domain.TailoringRequest) error
}

type AdminService struct {
	store AdminStore
	now   func() time.Time
}

func NewAdminService(store AdminStore) *AdminService {
	return &AdminService{store: store, now: func() time.Time { return time.Now().UTC() }}
}

func validID(id string) bool {
	if len(id) != 24 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

// infra wraps non-business errors so handlers can map them to 503.
func infra(err error) error {
	if err == nil {
		return nil
	}
	for _, b := range []error{ErrNotFound, ErrStateConflict, ErrMeasurementLocked, ErrMeasurementIncomplete} {
		if errors.Is(err, b) {
			return err
		}
	}
	var invalid *domain.InvalidFieldError
	if errors.As(err, &invalid) {
		return err
	}
	return fmt.Errorf("%w: %w", domain.ErrTailoringUnavailable, err)
}

func (s *AdminService) Get(ctx context.Context, id string) (*domain.TailoringRequest, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	r, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, infra(err)
	}
	if r == nil {
		return nil, ErrNotFound
	}
	return r, nil
}

func (s *AdminService) List(ctx context.Context, f AdminFilter) ([]domain.TailoringRequest, int64, int64, int64, error) {
	if f.Status != "" && !domain.ValidStatus(f.Status) {
		return nil, 0, 0, 0, domain.NewInvalidFieldError("status", "Trạng thái yêu cầu may đo không hợp lệ")
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	if f.Page-1 > math.MaxInt64/f.PageSize {
		return nil, 0, 0, 0, domain.NewInvalidFieldError("page", "Trang vượt giới hạn")
	}
	items, total, err := s.store.ListAdmin(ctx, f)
	if err != nil {
		return nil, 0, 0, 0, infra(err)
	}
	return items, total, f.Page, f.PageSize, nil
}

// mutate re-reads and re-evaluates on every attempt so two admins editing
// concurrently never overwrite each other with stale data.
func (s *AdminService) mutate(ctx context.Context, id string, apply func(cur *domain.TailoringRequest) (*domain.TailoringRequest, error)) (*domain.TailoringRequest, error) {
	for attempt := 0; attempt < maxWriteAttempts; attempt++ {
		cur, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		next, err := apply(cur)
		if err != nil {
			return nil, err
		}
		if next == nil { // no-op
			return cur, nil
		}
		now := s.now().Truncate(time.Millisecond)
		if !now.After(cur.UpdatedAt) {
			now = cur.UpdatedAt.Add(time.Millisecond)
		}
		next.UpdatedAt = now
		err = s.store.UpdateIf(ctx, cur, next)
		if err == nil {
			return next, nil
		}
		if !errors.Is(err, ErrConcurrentUpdate) {
			return nil, infra(err)
		}
	}
	return nil, ErrStateConflict
}

func (s *AdminService) Update(ctx context.Context, id string, in UpdateInput) (*domain.TailoringRequest, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	if in.empty() {
		return nil, domain.NewInvalidFieldError("body", "Phải cung cấp ít nhất một trường cần cập nhật")
	}
	// Validate input independent of stored state first, so bad input is 400
	// even when the document is locked.
	if in.Notes.Set && in.Notes.Value != nil {
		v, err := domain.ValidateNotes(in.Notes.Value)
		if err != nil {
			return nil, err
		}
		in.Notes.Value = v
	}
	if in.AssignedStaff.Set && in.AssignedStaff.Value != nil {
		t := strings.TrimSpace(*in.AssignedStaff.Value)
		if utf8.RuneCountInString(t) > MaxAssignedStaffLen {
			return nil, domain.NewInvalidFieldError("assigned_staff", "Người phụ trách không được vượt quá 200 ký tự")
		}
		if t == "" {
			in.AssignedStaff.Value = nil
		} else {
			in.AssignedStaff.Value = &t
		}
	}
	if in.PreferredContact.Set && in.PreferredContact.Value != nil {
		v, err := domain.ValidatePreferredContact(in.PreferredContact.Value)
		if err != nil {
			return nil, err
		}
		in.PreferredContact.Value = v
	}
	if in.AppointmentAt.Set && in.AppointmentAt.Value != nil {
		t := in.AppointmentAt.Value.UTC().Truncate(time.Millisecond)
		in.AppointmentAt.Value = &t
	}
	hasMeasurements := in.Measurements != nil && in.Measurements.any()

	return s.mutate(ctx, id, func(cur *domain.TailoringRequest) (*domain.TailoringRequest, error) {
		if hasMeasurements && domain.MeasurementsLocked(cur.Status) {
			return nil, ErrMeasurementLocked
		}
		next := *cur
		if in.Notes.Set {
			next.Notes = in.Notes.Value
		}
		if in.AppointmentAt.Set {
			next.AppointmentAt = in.AppointmentAt.Value
		}
		if in.AssignedStaff.Set {
			next.AssignedStaff = in.AssignedStaff.Value
		}
		if in.PreferredContact.Set {
			next.PreferredContact = in.PreferredContact.Value
		}
		if hasMeasurements {
			merged := domain.Measurements{}
			if cur.Measurements != nil {
				merged = *cur.Measurements
			}
			p := in.Measurements
			if p.Body.Set {
				merged.Body = p.Body.Value
			}
			for _, f := range []struct {
				o Optional[float64]
				d **float64
			}{
				{p.NeckCM, &merged.NeckCM}, {p.BustCM, &merged.BustCM}, {p.WaistCM, &merged.WaistCM},
				{p.HipsCM, &merged.HipsCM}, {p.ShoulderCM, &merged.ShoulderCM},
				{p.SleeveCM, &merged.SleeveCM}, {p.GarmentLengthCM, &merged.GarmentLengthCM},
			} {
				if f.o.Set {
					*f.d = f.o.Value
				}
			}
			v, err := domain.ValidateMeasurements(&merged)
			if err != nil {
				return nil, err
			}
			next.Measurements = v
		}
		return &next, nil
	})
}

func (s *AdminService) SetStatus(ctx context.Context, id, target string) (*domain.TailoringRequest, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	if !domain.ValidStatus(target) {
		return nil, domain.NewInvalidFieldError("status", "Trạng thái yêu cầu may đo không hợp lệ")
	}
	if target == domain.StatusMeasurementConfirmed {
		return nil, domain.NewInvalidFieldError("status", "measurement_confirmed chỉ được đặt qua thao tác confirm-measurements")
	}
	return s.mutate(ctx, id, func(cur *domain.TailoringRequest) (*domain.TailoringRequest, error) {
		if cur.Status == target {
			return nil, nil
		}
		if !domain.NextStatusAllowed(cur.Status, target) {
			return nil, ErrStateConflict
		}
		next := *cur
		next.Status = target
		return &next, nil
	})
}

func (s *AdminService) ConfirmMeasurements(ctx context.Context, id string) (*domain.TailoringRequest, error) {
	if !validID(id) {
		return nil, ErrNotFound
	}
	return s.mutate(ctx, id, func(cur *domain.TailoringRequest) (*domain.TailoringRequest, error) {
		if cur.Status == domain.StatusMeasurementConfirmed {
			return nil, nil // idempotent retry
		}
		if !domain.CanConfirmMeasurements(cur.Status) {
			return nil, ErrStateConflict
		}
		if !domain.HasAllCMMeasurements(cur.Measurements) {
			return nil, ErrMeasurementIncomplete
		}
		next := *cur
		next.Status = domain.StatusMeasurementConfirmed
		return &next, nil
	})
}
