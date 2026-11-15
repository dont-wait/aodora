package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	user      User
	password  string
	session   string
	expiresAt time.Time
	created   bool
	deleted   bool
}

func (f *fakeRepository) CreateCustomer(_ context.Context, input RegisterInput, passwordHash string) (User, error) {
	f.password = passwordHash
	f.created = true
	f.user = User{ID: "customer-id", FullName: input.FullName, Email: input.Email, Role: "customer"}
	return f.user, nil
}

func (f *fakeRepository) FindCustomerByEmail(context.Context, string) (User, string, error) {
	if f.user.ID == "" {
		return User{}, "", ErrUserNotFound
	}
	return f.user, f.password, nil
}

func (f *fakeRepository) CreateSession(_ context.Context, _ string, tokenHash string, expiresAt time.Time) error {
	f.session, f.expiresAt = tokenHash, expiresAt
	return nil
}

func (f *fakeRepository) FindSessionUser(_ context.Context, tokenHash string, now time.Time) (User, error) {
	if tokenHash != f.session || !f.expiresAt.After(now) {
		return User{}, ErrSessionNotFound
	}
	return f.user, nil
}

func (f *fakeRepository) DeleteSession(context.Context, string) error {
	f.deleted = true
	return nil
}

func TestRegisterNormalizesEmailAndHashesPassword(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, time.Hour)
	user, err := service.Register(context.Background(), RegisterInput{FullName: " Aodora ", Email: " Owner@Example.COM ", Password: "password-123"})
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "owner@example.com" || !repo.created {
		t.Fatalf("unexpected user: %+v", user)
	}
	if repo.password == "password-123" || repo.password == "" {
		t.Fatal("password was not hashed")
	}
}

func TestLoginCreatesSessionAndAuthenticates(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, time.Hour)
	_, err := service.Register(context.Background(), RegisterInput{FullName: "Owner", Email: "owner@example.com", Password: "password-123"})
	if err != nil {
		t.Fatal(err)
	}
	user, token, expiresAt, err := service.Login(context.Background(), "OWNER@example.com", "password-123")
	if err != nil {
		t.Fatal(err)
	}
	if user.ID == "" || token == "" || !expiresAt.After(time.Now()) {
		t.Fatalf("invalid login result: %+v, %q, %v", user, token, expiresAt)
	}
	authenticated, err := service.Authenticate(context.Background(), token)
	if err != nil || authenticated.ID != user.ID {
		t.Fatalf("authenticate = %+v, %v", authenticated, err)
	}
}

func TestLoginHidesCredentialFailures(t *testing.T) {
	service := NewService(&fakeRepository{}, time.Hour)
	_, _, _, err := service.Login(context.Background(), "missing@example.com", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v, want invalid credentials", err)
	}
}

func TestLogoutDeletesSession(t *testing.T) {
	repo := &fakeRepository{}
	service := NewService(repo, time.Hour)
	if err := service.Logout(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
	if !repo.deleted {
		t.Fatal("session was not deleted")
	}
}
