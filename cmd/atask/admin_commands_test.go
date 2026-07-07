package main

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/atask/atask/internal/auth"
)

// TestValidateUserExists exercises validateUserExists against a real
// PocketBase test app (tests.NewTestApp seeds a `users` auth collection with
// a test@example.com record — see internal/auth/pocketbase_test.go for the
// same pattern). This is the P1 fix's core guard: `atask admin assign-data
// --to <id>` must reject an --to that does not resolve to a real user
// *before* any row in atask.db is touched, otherwise a typo'd ID silently
// corrupts the orphan data (see the RunE comment in admin_commands.go for the
// full failure scenario).
func TestValidateUserExists(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	adapter := auth.NewPBAdapterFromApp(app)

	t.Run("real user passes", func(t *testing.T) {
		user, err := app.FindAuthRecordByEmail("users", "test@example.com")
		if err != nil {
			t.Fatalf("find seeded test user: %v", err)
		}
		if err := validateUserExists(adapter, user.Id); err != nil {
			t.Errorf("expected real user %q to validate, got error: %v", user.Id, err)
		}
	})

	t.Run("nonexistent user is rejected", func(t *testing.T) {
		err := validateUserExists(adapter, "definitely-not-a-real-user-id-0000000000")
		if err == nil {
			t.Fatal("expected an error for a nonexistent user ID, got nil")
		}
	})

	t.Run("empty string is rejected", func(t *testing.T) {
		if err := validateUserExists(adapter, ""); err == nil {
			t.Fatal("expected an error for an empty user ID, got nil")
		}
	})
}
