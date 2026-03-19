package domain_test

import (
	"testing"

	"github.com/atask/atask/internal/domain"
)

// --- Status tests ---

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   domain.Status
		expected string
	}{
		{domain.StatusPending, "pending"},
		{domain.StatusCompleted, "completed"},
		{domain.StatusCancelled, "cancelled"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected domain.Status
		wantErr  bool
	}{
		{"pending", domain.StatusPending, false},
		{"completed", domain.StatusCompleted, false},
		{"cancelled", domain.StatusCancelled, false},
		{"unknown", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domain.ParseStatus(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseStatus(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseStatus(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseStatus(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

// --- Schedule tests ---

func TestScheduleString(t *testing.T) {
	tests := []struct {
		schedule domain.Schedule
		expected string
	}{
		{domain.ScheduleInbox, "inbox"},
		{domain.ScheduleAnytime, "anytime"},
		{domain.ScheduleSomeday, "someday"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.schedule.String(); got != tt.expected {
				t.Errorf("Schedule.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input    string
		expected domain.Schedule
		wantErr  bool
	}{
		{"inbox", domain.ScheduleInbox, false},
		{"anytime", domain.ScheduleAnytime, false},
		{"someday", domain.ScheduleSomeday, false},
		{"unknown", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := domain.ParseSchedule(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseSchedule(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseSchedule(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseSchedule(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
