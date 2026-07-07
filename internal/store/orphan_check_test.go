package store

import (
	"context"
	"testing"
)

// newTestDB opens an in-memory SQLite database, applies all migrations, and
// registers a cleanup to close it.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB(:memory:): %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return db
}

func TestOrphanCounts_EmptyDB(t *testing.T) {
	db := newTestDB(t)
	counts, err := OrphanCounts(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 0 {
		t.Errorf("expected zero orphans on fresh DB, got %v", counts)
	}
}

func TestOrphanCounts_DetectsPreMultiUserData(t *testing.T) {
	db := newTestDB(t)
	_, err := db.DB.Exec(`INSERT INTO tasks (id, user_id, title, "index", today_index, created_at, updated_at) VALUES ('t1', '', 'orphan task', 0, 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert orphan task: %v", err)
	}

	counts, err := OrphanCounts(context.Background(), db.DB)
	if err != nil {
		t.Fatal(err)
	}
	if counts["tasks"] != 1 {
		t.Errorf("expected 1 orphaned task, got %v", counts["tasks"])
	}
}
