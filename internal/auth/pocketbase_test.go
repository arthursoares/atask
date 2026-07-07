package auth

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// newTestAdapter boots PocketBase's embedded test app (which seeds a `users`
// auth collection and a `_superusers` collection, each with a test@example.com
// record whose password is "1234567890") and wraps it in a PBAdapter.
func newTestAdapter(t *testing.T) (*PBAdapter, *tests.TestApp) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	return &PBAdapter{app: app}, app
}

// Binding requirement #4: ValidateToken must only accept tokens whose auth
// record belongs to the `users` collection. A _superusers (admin) token must
// NOT validate as an application user.
func TestValidateToken_RejectsSuperuserToken(t *testing.T) {
	adapter, app := newTestAdapter(t)

	// A valid users-collection token is accepted.
	user, err := app.FindAuthRecordByEmail("users", "test@example.com")
	if err != nil {
		t.Fatalf("find user: %v", err)
	}
	userToken, err := user.NewAuthToken()
	if err != nil {
		t.Fatalf("user NewAuthToken: %v", err)
	}
	gotID, err := adapter.ValidateToken(userToken)
	if err != nil {
		t.Fatalf("expected users token to validate, got %v", err)
	}
	if gotID != user.Id {
		t.Errorf("expected user id %q, got %q", user.Id, gotID)
	}

	// A _superusers token must be rejected.
	su, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "test@example.com")
	if err != nil {
		t.Fatalf("find superuser: %v", err)
	}
	suToken, err := su.NewAuthToken()
	if err != nil {
		t.Fatalf("superuser NewAuthToken: %v", err)
	}
	if _, err := adapter.ValidateToken(suToken); err == nil {
		t.Fatal("expected superuser token to be rejected by ValidateToken, got nil error")
	}
}

// Binding requirement #2: AuthWithPassword must not leak account existence.
// The not-found path and the wrong-password path must return the identical
// error, and a correct password must succeed.
func TestAuthWithPassword_NoEnumerationOracle(t *testing.T) {
	adapter, _ := newTestAdapter(t)

	_, _, notFoundErr := adapter.AuthWithPassword("does-not-exist@example.com", "whatever")
	if notFoundErr == nil {
		t.Fatal("expected error for unknown email")
	}

	_, _, wrongPassErr := adapter.AuthWithPassword("test@example.com", "wrong-password")
	if wrongPassErr == nil {
		t.Fatal("expected error for wrong password")
	}

	if notFoundErr.Error() != wrongPassErr.Error() {
		t.Errorf("enumeration oracle: not-found error %q != wrong-password error %q",
			notFoundErr.Error(), wrongPassErr.Error())
	}

	token, user, err := adapter.AuthWithPassword("test@example.com", "1234567890")
	if err != nil {
		t.Fatalf("expected valid login to succeed, got %v", err)
	}
	if token == "" {
		t.Error("expected a non-empty token on successful login")
	}
	if user == nil || user.Email != "test@example.com" {
		t.Errorf("expected user test@example.com, got %+v", user)
	}
}
