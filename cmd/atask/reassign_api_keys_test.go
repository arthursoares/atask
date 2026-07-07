package main

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/store"
)

// Codex P1 follow-up: migration 006 preserved existing api_keys.user_id as
// the legacy `users.id` (now a dropped table), so on an upgraded DB every
// pre-existing API key 401s (requireAuth's FindUserByID can't resolve the
// legacy ID). The documented `assign-data --to <userID>` remediation only
// ever updated `user_id = ''` across store.OrphanableTables and never
// touched api_keys, so those keys were permanently unusable even after
// running assign-data.
//
// reassignOrphanedAPIKeys is the factored-out, directly-testable retarget
// logic (admin_commands.go): given a transaction and a real AuthProvider, it
// reassigns every api_keys row whose user_id is either empty or does not
// resolve to a live PocketBase user, and leaves rows already pointing at a
// valid user untouched.

// insertAPIKeyRow inserts a minimal api_keys row directly for test seeding —
// mirrors the shape AuthService.CreateAPIKey / migration 006 produce, without
// pulling in the sqlc query for a scope no test here cares about.
func insertAPIKeyRow(t *testing.T, db *store.DB, id, userID string) {
	t.Helper()
	_, err := db.DB.Exec(
		`INSERT INTO api_keys (id, user_id, name, key_hash, permissions, scope, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, userID, "test-key-"+id, id+"-hash", "[]", "read_write", time.Now(),
	)
	if err != nil {
		t.Fatalf("insert api_keys row %s: %v", id, err)
	}
}

func apiKeyUserID(t *testing.T, db *store.DB, id string) string {
	t.Helper()
	var userID string
	if err := db.DB.QueryRow(`SELECT user_id FROM api_keys WHERE id = ?`, id).Scan(&userID); err != nil {
		t.Fatalf("query api_keys.user_id for %s: %v", id, err)
	}
	return userID
}

func TestReassignOrphanedAPIKeys(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	adapter := auth.NewPBAdapterFromApp(app)

	target, err := adapter.CreateUser("reassign-target@example.com", "targetpass1", "Target", "user")
	if err != nil {
		t.Fatalf("create target user: %v", err)
	}
	liveUser, err := adapter.CreateUser("reassign-live@example.com", "livepass1", "Live", "user")
	if err != nil {
		t.Fatalf("create live user: %v", err)
	}

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("store.NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	insertAPIKeyRow(t, db, "key-empty", "")
	insertAPIKeyRow(t, db, "key-dangling", "dangling-legacy-id")
	insertAPIKeyRow(t, db, "key-valid", liveUser.ID)

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	n, err := reassignOrphanedAPIKeys(tx, adapter, target.ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("reassignOrphanedAPIKeys: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	if n != 2 {
		t.Errorf("expected 2 rows reassigned, got %d", n)
	}
	if got := apiKeyUserID(t, db, "key-empty"); got != target.ID {
		t.Errorf("key-empty: expected user_id %q, got %q", target.ID, got)
	}
	if got := apiKeyUserID(t, db, "key-dangling"); got != target.ID {
		t.Errorf("key-dangling: expected user_id %q, got %q", target.ID, got)
	}
	if got := apiKeyUserID(t, db, "key-valid"); got != liveUser.ID {
		t.Errorf("key-valid: expected user_id to remain %q (untouched), got %q", liveUser.ID, got)
	}
}

// TestReassignOrphanedAPIKeys_NoOrphans confirms a clean DB (every api_keys
// row already pointing at a live user) reports zero reassignments and leaves
// every row untouched.
func TestReassignOrphanedAPIKeys_NoOrphans(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)
	adapter := auth.NewPBAdapterFromApp(app)

	liveUser, err := adapter.CreateUser("reassign-clean@example.com", "cleanpass1", "Clean", "user")
	if err != nil {
		t.Fatalf("create live user: %v", err)
	}

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("store.NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	insertAPIKeyRow(t, db, "key-clean", liveUser.ID)

	tx, err := db.DB.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	n, err := reassignOrphanedAPIKeys(tx, adapter, liveUser.ID)
	if err != nil {
		tx.Rollback()
		t.Fatalf("reassignOrphanedAPIKeys: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit tx: %v", err)
	}

	if n != 0 {
		t.Errorf("expected 0 rows reassigned on a clean DB, got %d", n)
	}
	if got := apiKeyUserID(t, db, "key-clean"); got != liveUser.ID {
		t.Errorf("key-clean: expected user_id to remain %q, got %q", liveUser.ID, got)
	}
}
