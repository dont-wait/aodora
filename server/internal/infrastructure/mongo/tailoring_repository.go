package mongo

import (
	"context"
	"errors"
	"time"

	apptailoring "github.com/your-org/aodora/server/internal/application/tailoring"
	domaintailoring "github.com/your-org/aodora/server/internal/domain/tailoring"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
)

type TailoringRepository struct {
	requests *driver.Collection
	products *driver.Collection
}

func NewTailoringRepository(db *driver.Database) *TailoringRepository {
	return &TailoringRepository{
		requests: db.Collection("tailoring_requests"),
		products: db.Collection("products"),
	}
}

type storedMeasurements struct {
	Body            *string  `bson:"body,omitempty"`
	NeckCM          *float64 `bson:"neck_cm,omitempty"`
	BustCM          *float64 `bson:"bust_cm,omitempty"`
	WaistCM         *float64 `bson:"waist_cm,omitempty"`
	HipsCM          *float64 `bson:"hips_cm,omitempty"`
	ShoulderCM      *float64 `bson:"shoulder_cm,omitempty"`
	SleeveCM        *float64 `bson:"sleeve_cm,omitempty"`
	GarmentLengthCM *float64 `bson:"garment_length_cm,omitempty"`
}

type storedTailoringRequest struct {
	ID               bson.ObjectID       `bson:"_id"`
	CustomerID       *bson.ObjectID      `bson:"customer_id,omitempty"`
	FullName         string              `bson:"full_name"`
	Phone            string              `bson:"phone"`
	PreferredContact *string             `bson:"preferred_contact,omitempty"`
	CategoryID       *string             `bson:"category_id,omitempty"`
	ProductID        *bson.ObjectID      `bson:"product_id,omitempty"`
	ProductSlug      *string             `bson:"product_slug,omitempty"`
	Measurements     *storedMeasurements `bson:"measurements,omitempty"`
	Notes            *string             `bson:"notes,omitempty"`
	Status           string              `bson:"status"`
	AppointmentAt    *time.Time          `bson:"appointment_at,omitempty"`
	AssignedStaff    *string             `bson:"assigned_staff,omitempty"`
	CreatedAt        time.Time           `bson:"created_at"`
	UpdatedAt        time.Time           `bson:"updated_at"`
}

func (r *TailoringRepository) Create(ctx context.Context, req *domaintailoring.TailoringRequest) (*domaintailoring.TailoringRequest, error) {
	docID := bson.NewObjectID()
	if req.ID != "" {
		if parsed, err := bson.ObjectIDFromHex(req.ID); err == nil {
			docID = parsed
		}
	}

	var customerOID *bson.ObjectID
	if req.CustomerID != nil && *req.CustomerID != "" {
		if parsed, err := bson.ObjectIDFromHex(*req.CustomerID); err == nil {
			customerOID = &parsed
		}
	}

	var productOID *bson.ObjectID
	if req.ProductID != nil && *req.ProductID != "" {
		if parsed, err := bson.ObjectIDFromHex(*req.ProductID); err == nil {
			productOID = &parsed
		}
	}

	var measurements *storedMeasurements
	if req.Measurements != nil {
		measurements = &storedMeasurements{
			Body:            req.Measurements.Body,
			NeckCM:          req.Measurements.NeckCM,
			BustCM:          req.Measurements.BustCM,
			WaistCM:         req.Measurements.WaistCM,
			HipsCM:          req.Measurements.HipsCM,
			ShoulderCM:      req.Measurements.ShoulderCM,
			SleeveCM:        req.Measurements.SleeveCM,
			GarmentLengthCM: req.Measurements.GarmentLengthCM,
		}
	}

	doc := storedTailoringRequest{
		ID:               docID,
		CustomerID:       customerOID,
		FullName:         req.FullName,
		Phone:            req.Phone,
		PreferredContact: req.PreferredContact,
		CategoryID:       req.CategoryID,
		ProductID:        productOID,
		ProductSlug:      req.ProductSlug,
		Measurements:     measurements,
		Notes:            req.Notes,
		Status:           req.Status,
		AppointmentAt:    req.AppointmentAt,
		AssignedStaff:    req.AssignedStaff,
		CreatedAt:        req.CreatedAt,
		UpdatedAt:        req.UpdatedAt,
	}

	if _, err := r.requests.InsertOne(ctx, doc); err != nil {
		return nil, err
	}

	out := *req
	out.ID = docID.Hex()
	return &out, nil
}

func (r *TailoringRepository) FindProductByID(ctx context.Context, id string) (*apptailoring.ProductRef, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var doc struct {
		ID         bson.ObjectID `bson:"_id"`
		Slug       string        `bson:"slug"`
		CategoryID any           `bson:"category_id"`
		Name       string        `bson:"name"`
		IsActive   bool          `bson:"is_active"`
	}

	if err := r.products.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, domaintailoring.ErrTailoringUnavailable
	}

	var catID string
	switch v := doc.CategoryID.(type) {
	case bson.ObjectID:
		catID = v.Hex()
	case string:
		catID = v
	}

	return &apptailoring.ProductRef{
		ID:         doc.ID.Hex(),
		Slug:       doc.Slug,
		CategoryID: catID,
		Name:       doc.Name,
		IsActive:   doc.IsActive,
	}, nil
}

func (r *TailoringRepository) FindProductBySlug(ctx context.Context, slug string) (*apptailoring.ProductRef, error) {
	var doc struct {
		ID         bson.ObjectID `bson:"_id"`
		Slug       string        `bson:"slug"`
		CategoryID any           `bson:"category_id"`
		Name       string        `bson:"name"`
		IsActive   bool          `bson:"is_active"`
	}

	if err := r.products.FindOne(ctx, bson.M{"slug": slug}).Decode(&doc); err != nil {
		if errors.Is(err, driver.ErrNoDocuments) {
			return nil, nil
		}
		return nil, domaintailoring.ErrTailoringUnavailable
	}

	var catID string
	switch v := doc.CategoryID.(type) {
	case bson.ObjectID:
		catID = v.Hex()
	case string:
		catID = v
	}

	return &apptailoring.ProductRef{
		ID:         doc.ID.Hex(),
		Slug:       doc.Slug,
		CategoryID: catID,
		Name:       doc.Name,
		IsActive:   doc.IsActive,
	}, nil
}
