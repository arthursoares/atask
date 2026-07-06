package service

import (
	"context"
	"errors"
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

// TestAuthService_CreateUser: legacy auth removed in Task 1.5; rewritten in Task 12.
// CreateUser now always returns errLegacyAuthRemoved (the `users` table was dropped
// in migration 006). This test exercised exactly that removed behavior.
func TestAuthService_CreateUser(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.CreateUser(ctx, "alice@example.com", "password123", "Alice"); !errors.Is(err, errLegacyAuthRemoved) {
		t.Fatalf("expected errLegacyAuthRemoved, got %v", err)
	}
}

// TestAuthService_Login: legacy auth removed in Task 1.5; rewritten in Task 12.
// Login now always returns errLegacyAuthRemoved (the `users` table was dropped
// in migration 006). This test exercised exactly that removed behavior.
func TestAuthService_Login(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	if _, err := svc.Login(ctx, "bob@example.com", "secret"); !errors.Is(err, errLegacyAuthRemoved) {
		t.Fatalf("expected errLegacyAuthRemoved, got %v", err)
	}
}

// TestAuthService_ValidateToken is skipped: legacy auth removed in Task 1.5;
// rewritten in Task 12. The only way to mint a valid signed token was through the
// now-stubbed Login method (which depended on the dropped `users` table); Task 12's
// AuthProvider will supply a new way to produce a token for ValidateToken to check.
func TestAuthService_ValidateToken(t *testing.T) {
	t.Skip("legacy auth removed in Task 1.5; rewritten in Task 12")
}

func TestAuthService_CreateAndValidateAPIKey(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	// API keys no longer reference the (now-dropped) users table, so a literal
	// user ID is sufficient here instead of provisioning a user via CreateUser.
	userID := "user-dave"

	plainKey, apiKey, err := svc.CreateAPIKey(ctx, userID, "my-key")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	if plainKey == "" {
		t.Error("expected non-empty plaintext key")
	}
	if apiKey.ID == "" {
		t.Error("expected non-empty API key ID")
	}
	if apiKey.UserID != userID {
		t.Errorf("expected userID %q, got %q", userID, apiKey.UserID)
	}

	gotUserID, gotKeyID, err := svc.ValidateAPIKey(ctx, plainKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey: %v", err)
	}
	if gotUserID != userID {
		t.Errorf("expected userID %q, got %q", userID, gotUserID)
	}
	if gotKeyID != apiKey.ID {
		t.Errorf("expected keyID %q, got %q", apiKey.ID, gotKeyID)
	}
}

func TestAuthService_ListAPIKeys(t *testing.T) {
	svc := newTestAuthService(t)
	ctx := context.Background()

	// API keys no longer reference the (now-dropped) users table, so a literal
	// user ID is sufficient here instead of provisioning a user via CreateUser.
	userID := "user-eve"

	_, _, err := svc.CreateAPIKey(ctx, userID, "key-one")
	if err != nil {
		t.Fatalf("CreateAPIKey 1: %v", err)
	}
	_, _, err = svc.CreateAPIKey(ctx, userID, "key-two")
	if err != nil {
		t.Fatalf("CreateAPIKey 2: %v", err)
	}

	keys, err := svc.ListAPIKeys(ctx, userID)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}
