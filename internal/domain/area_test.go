package domain

import "testing"

func TestNewArea(t *testing.T) {
	a, err := NewArea("Health")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if a.ID == "" {
		t.Error("expected non-empty ID")
	}

	if a.Title != "Health" {
		t.Errorf("expected title %q, got %q", "Health", a.Title)
	}

	if a.Archived {
		t.Error("expected Archived to be false by default")
	}

	if a.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if a.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewArea_EmptyTitle(t *testing.T) {
	_, err := NewArea("")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}
