package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	app "github.com/your-org/aodora/server/internal/application/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
	driver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type AuthRepository struct {
	customers *driver.Collection
	sessions  *driver.Collection
}

func NewAuthRepository(db *driver.Database) *AuthRepository {
	return &AuthRepository{customers: db.Collection("customers"), sessions: db.Collection("sessions")}
}

type customerDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	FullName     string        `bson:"full_name"`
	Email        string        `bson:"email"`
	EmailNorm    string        `bson:"email_normalized"`
	Phone        string        `bson:"phone,omitempty"`
	PasswordHash string        `bson:"password_hash"`
	Role         string        `bson:"role,omitempty"`
}

func userFromDocument(doc customerDocument) app.User {
	role := doc.Role
	if role == "" {
		role = "customer"
	}
	return app.User{ID: doc.ID.Hex(), FullName: doc.FullName, Email: doc.Email, Phone: doc.Phone, Role: role}
}

func (r *AuthRepository) CreateCustomer(ctx context.Context, input app.RegisterInput, passwordHash string) (app.User, error) {
	now := time.Now().UTC()
	doc := customerDocument{
		ID:           bson.NewObjectID(),
		FullName:     input.FullName,
		Email:        input.Email,
		EmailNorm:    input.Email,
		Phone:        input.Phone,
		PasswordHash: passwordHash,
		Role:         "customer",
	}
	_, err := r.customers.InsertOne(ctx, bson.M{
		"_id": doc.ID, "full_name": doc.FullName, "email": doc.Email, "email_normalized": doc.EmailNorm,
		"password_hash": doc.PasswordHash, "role": doc.Role, "phone": doc.Phone,
		"created_at": now, "updated_at": now,
	})
	if driver.IsDuplicateKeyError(err) {
		return app.User{}, app.ErrEmailTaken
	}
	if err != nil {
		return app.User{}, fmt.Errorf("create customer: %w", err)
	}
	return userFromDocument(doc), nil
}

func (r *AuthRepository) FindCustomerByEmail(ctx context.Context, email string) (app.User, string, error) {
	var doc customerDocument
	filter := bson.M{"$or": bson.A{
		bson.M{"email_normalized": email},
		bson.M{"email": email},
	}}
	err := r.customers.FindOne(ctx, filter, options.FindOne().SetCollation(&options.Collation{Locale: "en", Strength: 2})).Decode(&doc)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.User{}, "", app.ErrUserNotFound
	}
	if err != nil {
		return app.User{}, "", fmt.Errorf("find customer: %w", err)
	}
	return userFromDocument(doc), doc.PasswordHash, nil
}

type sessionDocument struct {
	CustomerID bson.ObjectID `bson:"customer_id"`
	ExpiresAt  time.Time     `bson:"expires_at"`
}

func (r *AuthRepository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	id, err := bson.ObjectIDFromHex(userID)
	if err != nil {
		return fmt.Errorf("create session: invalid customer id: %w", err)
	}
	_, err = r.sessions.InsertOne(ctx, bson.M{
		"token_hash": tokenHash, "customer_id": id,
		"created_at": time.Now().UTC(), "expires_at": expiresAt,
	})
	return err
}

func (r *AuthRepository) FindSessionUser(ctx context.Context, tokenHash string, now time.Time) (app.User, error) {
	var session sessionDocument
	err := r.sessions.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&session)
	if errors.Is(err, driver.ErrNoDocuments) {
		return app.User{}, app.ErrSessionNotFound
	}
	if err != nil {
		return app.User{}, fmt.Errorf("find session: %w", err)
	}
	if !session.ExpiresAt.After(now) {
		_, _ = r.sessions.DeleteOne(ctx, bson.M{"token_hash": tokenHash})
		return app.User{}, app.ErrSessionNotFound
	}
	var customer customerDocument
	if err := r.customers.FindOne(ctx, bson.M{"_id": session.CustomerID}).Decode(&customer); errors.Is(err, driver.ErrNoDocuments) {
		return app.User{}, app.ErrUserNotFound
	} else if err != nil {
		return app.User{}, fmt.Errorf("find session customer: %w", err)
	}
	return userFromDocument(customer), nil
}

func (r *AuthRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.sessions.DeleteOne(ctx, bson.M{"token_hash": tokenHash})
	return err
}

func (r *AuthRepository) UpsertAdmin(ctx context.Context, email, fullName, phone, passwordHash string) error {
	now := time.Now().UTC()
	_, err := r.customers.UpdateOne(ctx, bson.M{"$or": bson.A{
		bson.M{"email_normalized": email}, bson.M{"email": email},
	}}, bson.M{
		"$set":         bson.M{"full_name": fullName, "email": email, "email_normalized": email, "phone": phone, "password_hash": passwordHash, "role": "admin", "updated_at": now},
		"$setOnInsert": bson.M{"created_at": now},
	}, options.UpdateOne().SetUpsert(true).SetCollation(&options.Collation{Locale: "en", Strength: 2}))
	if driver.IsDuplicateKeyError(err) {
		return app.ErrEmailTaken
	}
	return err
}
