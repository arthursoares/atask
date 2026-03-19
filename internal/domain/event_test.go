package domain

import (
	"testing"
)

func TestNewDomainEvent(t *testing.T) {
	payload := map[string]any{"title": "Buy milk"}
	event := NewDomainEvent(TaskCreated, "task", "task-123", "user-456", payload)

	if event.Type != TaskCreated {
		t.Errorf("expected Type %q, got %q", TaskCreated, event.Type)
	}
	if event.EntityID != "task-123" {
		t.Errorf("expected EntityID %q, got %q", "task-123", event.EntityID)
	}
	if event.EntityType != "task" {
		t.Errorf("expected EntityType %q, got %q", "task", event.EntityType)
	}
	if event.ActorID != "user-456" {
		t.Errorf("expected ActorID %q, got %q", "user-456", event.ActorID)
	}
	if event.Payload["title"] != "Buy milk" {
		t.Errorf("expected Payload[title] %q, got %v", "Buy milk", event.Payload["title"])
	}
	if event.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set, got zero value")
	}
}

func TestDeltaAction_String(t *testing.T) {
	tests := []struct {
		action   DeltaAction
		expected string
	}{
		{DeltaCreated, "created"},
		{DeltaModified, "modified"},
		{DeltaDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.action.String()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
