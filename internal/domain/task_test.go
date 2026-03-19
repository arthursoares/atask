package domain

import (
	"testing"
)

func TestNewTask(t *testing.T) {
	task, err := NewTask("Buy groceries")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if task.ID == "" {
		t.Error("expected non-empty ID")
	}

	if task.Title != "Buy groceries" {
		t.Errorf("expected title %q, got %q", "Buy groceries", task.Title)
	}

	if task.Status != StatusPending {
		t.Errorf("expected status %v, got %v", StatusPending, task.Status)
	}

	if task.Schedule != ScheduleInbox {
		t.Errorf("expected schedule %v, got %v", ScheduleInbox, task.Schedule)
	}

	if task.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}

	if task.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewTask_EmptyTitle(t *testing.T) {
	_, err := NewTask("")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
}

func TestTask_Validate(t *testing.T) {
	t.Run("valid standalone task", func(t *testing.T) {
		task, err := NewTask("Standalone task")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := task.Validate(); err != nil {
			t.Errorf("expected no validation error, got: %v", err)
		}
	})

	t.Run("valid task with project", func(t *testing.T) {
		task, err := NewTask("Task with project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		projectID := "proj-123"
		sectionID := "sec-456"
		task.ProjectID = &projectID
		task.SectionID = &sectionID
		if err := task.Validate(); err != nil {
			t.Errorf("expected no validation error, got: %v", err)
		}
	})

	t.Run("invalid section without project", func(t *testing.T) {
		task, err := NewTask("Task with section but no project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sectionID := "sec-456"
		task.SectionID = &sectionID
		if err := task.Validate(); err == nil {
			t.Error("expected validation error for section without project, got nil")
		}
	})
}
