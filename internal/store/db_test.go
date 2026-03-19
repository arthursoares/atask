package store

import (
	"context"
	"testing"
)

func TestNewDB_InMemory(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB(:memory:) error: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping() error: %v", err)
	}
}

func TestNewDB_RunMigrations(t *testing.T) {
	db, err := NewDB(":memory:")
	if err != nil {
		t.Fatalf("NewDB(:memory:) error: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error: %v", err)
	}

	// Verify "tasks" table exists
	var name string
	row := db.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='tasks'`)
	if err := row.Scan(&name); err != nil {
		t.Fatalf("tasks table not found after migration: %v", err)
	}
	if name != "tasks" {
		t.Errorf("expected table name 'tasks', got %q", name)
	}
}
