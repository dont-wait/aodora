package mongo

import (
	"context"
	"errors"

	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (d storedTailoringRequest) toDomain() domaintailoring.TailoringRequest {
	out := domaintailoring.TailoringRequest{
		ID:               d.ID.Hex(),
		FullName:         d.FullName,
		Phone:            d.Phone,
		PreferredContact: d.PreferredContact,
		CategoryID:       d.CategoryID,
		ProductSlug:      d.ProductSlug,
		Notes:            d.Notes,
		Status:           d.Status,
		AppointmentAt:    d.AppointmentAt,
		AssignedStaff:    d.AssignedStaff,
		CreatedAt:        d.CreatedAt,
		UpdatedAt:        d.UpdatedAt,
	}
	if d.CustomerID != nil {
		h := d.CustomerID.Hex()
		out.CustomerID = &h
	}
	if d.ProductID != nil {
		h := d.ProductID.Hex()
		out.ProductID = &h
	}
	if m := d.Measurements; m != nil {
		out.Measurements = &domaintailoring.Measurements{
			Body: m.Body, NeckCM: m.NeckCM, BustCM: m.BustCM, WaistCM: m.WaistCM, HipsCM: m.HipsCM,
			ShoulderCM: m.ShoulderCM, SleeveCM: m.SleeveCM, GarmentLengthCM: m.GarmentLengthCM,
		}
	}
	return out
}

// FindByID returns (nil, nil) when the document does not exist.
func (r *TailoringRepository) FindByID(ctx context.Context, id string) (*domaintailoring.TailoringRequest, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, nil
	}
	var doc storedTailoringRequest
	if err := r.requests.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	out := doc.toDomain()
	return &out, nil
}

// ListAdmin sorts by created_at desc then _id desc so pages are stable.
func (r *TailoringRepository) ListAdmin(ctx context.Context, f apptailoring.AdminFilter) ([]domaintailoring.TailoringRequest, int64, error) {
	filter := bson.M{}
	if f.Status != "" {
		filter["status"] = f.Status
	}
	total, err := r.requests.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	cur, err := r.requests.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip((f.Page-1)*f.PageSize).
		SetLimit(f.PageSize))
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	items := make([]domaintailoring.TailoringRequest, 0, f.PageSize)
	for cur.Next(ctx) {
		var doc storedTailoringRequest
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		items = append(items, doc.toDomain())
	}
	if err := cur.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateIf writes only the admin-editable fields, guarded by the status and
// updated_at that were read. Cleared fields are $unset (the schema does not
// accept null for preferred_contact or numeric measurements).
func (r *TailoringRepository) UpdateIf(ctx context.Context, cur, next *domaintailoring.TailoringRequest) error {
	oid, err := bson.ObjectIDFromHex(cur.ID)
	if err != nil {
		return apptailoring.ErrNotFound
	}
	set := bson.M{"status": next.Status, "updated_at": next.UpdatedAt}
	unset := bson.M{}
	put := func(key string, present bool, value any) {
		if present {
			set[key] = value
		} else {
			unset[key] = ""
		}
	}
	put("notes", next.Notes != nil, derefString(next.Notes))
	put("assigned_staff", next.AssignedStaff != nil, derefString(next.AssignedStaff))
	put("preferred_contact", next.PreferredContact != nil, derefString(next.PreferredContact))
	if next.AppointmentAt != nil {
		set["appointment_at"] = *next.AppointmentAt
	} else {
		unset["appointment_at"] = ""
	}
	if m := next.Measurements; m != nil {
		set["measurements"] = storedMeasurements{
			Body: m.Body, NeckCM: m.NeckCM, BustCM: m.BustCM, WaistCM: m.WaistCM, HipsCM: m.HipsCM,
			ShoulderCM: m.ShoulderCM, SleeveCM: m.SleeveCM, GarmentLengthCM: m.GarmentLengthCM,
		}
	} else {
		unset["measurements"] = ""
	}
	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	res, err := r.requests.UpdateOne(ctx, bson.M{"_id": oid, "status": cur.Status, "updated_at": cur.UpdatedAt}, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return apptailoring.ErrConcurrentUpdate
	}
	return nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
