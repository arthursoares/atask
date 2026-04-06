package domain

import "errors"

// RecurrenceMode defines how a recurring task is rescheduled.
type RecurrenceMode string

const (
	RecurrenceModeFixed           RecurrenceMode = "fixed"
	RecurrenceModeAfterCompletion RecurrenceMode = "afterCompletion"
)

// RecurrenceUnit defines the time unit for recurrence intervals.
type RecurrenceUnit string

const (
	RecurrenceUnitDay   RecurrenceUnit = "day"
	RecurrenceUnitWeek  RecurrenceUnit = "week"
	RecurrenceUnitMonth RecurrenceUnit = "month"
)

// RecurrenceEnd defines when a recurrence series terminates.
type RecurrenceEnd struct {
	Date  *string `json:"date,omitempty"`
	Count *int    `json:"count,omitempty"`
}

// RecurrenceRule describes the recurrence configuration for a task.
type RecurrenceRule struct {
	Mode     RecurrenceMode `json:"type"`
	Interval int            `json:"interval"`
	Unit     RecurrenceUnit `json:"unit"`
	End      *RecurrenceEnd `json:"end,omitempty"`
}

// Validate checks that the RecurrenceRule fields are valid.
func (r *RecurrenceRule) Validate() error {
	if r.Interval <= 0 {
		return errors.New("recurrence interval must be greater than zero")
	}

	switch r.Mode {
	case RecurrenceModeFixed, RecurrenceModeAfterCompletion:
		// valid
	default:
		return errors.New("recurrence mode must be \"fixed\" or \"afterCompletion\"")
	}

	switch r.Unit {
	case RecurrenceUnitDay, RecurrenceUnitWeek, RecurrenceUnitMonth:
		// valid
	default:
		return errors.New("recurrence unit must be \"day\", \"week\", or \"month\"")
	}

	return nil
}
