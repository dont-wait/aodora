//go:build integration

package mongo

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appauth "github.com/your-org/aodora/server/internal/application/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestAuthSessionEndsWhenRoleChanges(t *testing.T) {
	_, db, _ := orderTestDatabase(t)
	ctx := context.Background()
	repo := NewAuthRepository(db)
	now := time.Now().UTC()
	id := bson.NewObjectID()
	if _, err := db.Collection("customers").InsertOne(ctx, bson.M{
		"_id": id, "full_name": "Role Test", "email": "role-test@example.com", "email_normalized": "role-test@example.com",
		"password_hash": "$2a$10$abcdefghijklmnopqrstuv", "role": "admin", "created_at": now, "updated_at": now,
	}); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	hash := strings.Repeat("a", 64)
	if err := repo.CreateSession(ctx, id.Hex(), "admin", hash, now.Add(time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}
	var stored bson.M
	if err := db.Collection("sessions").FindOne(ctx, bson.M{"token_hash": hash}).Decode(&stored); err != nil || stored["role"] != "admin" {
		t.Fatalf("session role = %v, err %v; want admin", stored["role"], err)
	}
	if user, err := repo.FindSessionUser(ctx, hash, now); err != nil || user.Role != "admin" {
		t.Fatalf("FindSessionUser = %+v, %v; want admin", user, err)
	}
	if _, err := db.Collection("customers").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"role": "customer"}}); err != nil {
		t.Fatalf("demote: %v", err)
	}
	if _, err := repo.FindSessionUser(ctx, hash, now); !errors.Is(err, appauth.ErrSessionNotFound) {
		t.Fatalf("after demotion err = %v; want ErrSessionNotFound", err)
	}
	if n, _ := db.Collection("sessions").CountDocuments(ctx, bson.M{"token_hash": hash}); n != 0 {
		t.Fatalf("stale session kept: %d", n)
	}
}
