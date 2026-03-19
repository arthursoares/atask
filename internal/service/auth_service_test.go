package service

import (
	"context"
	"testing"

	"github.com/atask/atask/internal/store"
)

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	return NewAuthService(db, "test-secret-key")
}

func TestAuthService_CreateUser(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "alice@example.com", "password123", "Alice")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email %q, got %q", "alice@example.com", user.Email)
	}
	if user.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", user.Name)
	}
}

func TestAuthService_Login(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	_, err := svc.CreateUser(ctx, "bob@example.com", "secret", "Bob")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, err := svc.Login(ctx, "bob@example.com", "secret")
	if err != nil {
		t.Fatalf("Login with correct password: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	_, err = svc.Login(ctx, "bob@example.com", "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "carol@example.com", "pass", "Carol")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	token, err := svc.Login(ctx, "carol@example.com", "pass")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, userID)
	}
}

func TestAuthService_CreateAndValidateAPIKey(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "dave@example.com", "pass", "Dave")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	plainKey, apiKey, err := svc.CreateAPIKey(ctx, user.ID, "my-key")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if plainKey == "" {
		t.Error("expected non-empty plaintext key")
	}
	if apiKey.ID == "" {
		t.Error("expected non-empty API key ID")
	}
	if apiKey.UserID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, apiKey.UserID)
	}

	gotUserID, gotKeyID, err := svc.ValidateAPIKey(ctx, plainKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if gotUserID != user.ID {
		t.Errorf("expected userID %q, got %q", user.ID, gotUserID)
	}
	if gotKeyID != apiKey.ID {
		t.Errorf("expected keyID %q, got %q", apiKey.ID, gotKeyID)
	}
}

func TestAuthService_ListAPIKeys(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	user, err := svc.CreateUser(ctx, "eve@example.com", "pass", "Eve")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	_, _, err = svc.CreateAPIKey(ctx, user.ID, "key-one")
	if err != nil {
		t.Fatalf("CreateAPIKey 1: %v", err)
	}
	_, _, err = svc.CreateAPIKey(ctx, user.ID, "key-two")
	if err != nil {
		t.Fatalf("CreateAPIKey 2: %v", err)
	}

	keys, err := svc.ListAPIKeys(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}
