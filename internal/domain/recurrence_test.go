package domain

import "testing"

func TestRecurrenceRule_Validate(t *testing.T) {
	t.Run("valid fixed daily", func(t *testing.T) {
		r := &RecurrenceRule{
			Mode:     RecurrenceModeFixed,
			Interval: 1,
			Unit:     RecurrenceUnitDay,
		}
		if err := r.Validate(); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("valid after_completion weekly", func(t *testing.T) {
		r := &RecurrenceRule{
			Mode:     RecurrenceModeAfterCompletion,
			Interval: 2,
			Unit:     RecurrenceUnitWeek,
		}
		if err := r.Validate(); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("zero interval error", func(t *testing.T) {
		r := &RecurrenceRule{
			Mode:     RecurrenceModeFixed,
			Interval: 0,
			Unit:     RecurrenceUnitDay,
		}
		if err := r.Validate(); err == nil {
			t.Error("expected error for zero interval, got nil")
		}
	})

	t.Run("negative interval error", func(t *testing.T) {
		r := &RecurrenceRule{
			Mode:     RecurrenceModeFixed,
			Interval: -3,
			Unit:     RecurrenceUnitDay,
		}
		if err := r.Validate(); err == nil {
			t.Error("expected error for negative interval, got nil")
		}
	})
}
