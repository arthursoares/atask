package domain

import "testing"

func TestNewProject(t *testing.T) {
	p, err := NewProject("Launch website")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if p.ID == "" {
		t.Error("expected non-empty ID")
	}

	if p.Title != "Launch website" {
		t.Errorf("expected title %q, got %q", "Launch website", p.Title)
	}

	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if p.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewProject_EmptyTitle(t *testing.T) {
	_, err := NewProject("")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}
