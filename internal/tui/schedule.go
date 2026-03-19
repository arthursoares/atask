package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ScheduleSelectedMsg is sent when the user picks a schedule option.
type ScheduleSelectedMsg struct {
	TaskID   string
	Schedule string // "inbox", "anytime", or "someday"
}

// scheduleOptions are the human-readable labels shown in the picker.
var scheduleOptions = []string{"Inbox", "Today", "Someday"}

// scheduleValues maps each label to the API schedule value.
var scheduleValues = map[string]string{
	"Inbox":   "inbox",
	"Today":   "anytime",
	"Someday": "someday",
}

// Schedule is a small overlay that lets the user pick a schedule for a task.
type Schedule struct {
	taskID  string
	options []string
	cursor  int
	width   int
}

// NewSchedule creates a new Schedule picker for the given task.
func NewSchedule(taskID string, width int) Schedule {
	return Schedule{
		taskID:  taskID,
		options: scheduleOptions,
		cursor:  0,
		width:   width,
	}
}

// Update handles key messages. Returns the updated Schedule, a command, and
// whether the overlay was closed (true = Enter/Esc pressed).
func (s Schedule) Update(msg tea.KeyPressMsg) (Schedule, tea.Cmd, bool) {
	switch {
	case isRune(msg, 'j') || isDown(msg):
		if s.cursor < len(s.options)-1 {
			s.cursor++
		}
		return s, nil, false

	case isRune(msg, 'k') || isUp(msg):
		if s.cursor > 0 {
			s.cursor--
		}
		return s, nil, false

	case isEnter(msg):
		label := s.options[s.cursor]
		scheduleVal := scheduleValues[label]
		taskID := s.taskID
		cmd := func() tea.Msg {
			return ScheduleSelectedMsg{TaskID: taskID, Schedule: scheduleVal}
		}
		return s, cmd, true

	case isEscape(msg):
		return s, nil, true
	}

	return s, nil, false
}

// View renders the schedule picker content (wrapped by OverlayStyle in the parent).
func (s Schedule) View() string {
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("17")).
		Foreground(lipgloss.Cyan).
		Bold(true)

	var sb strings.Builder
	for i, opt := range s.options {
		if i == s.cursor {
			sb.WriteString(selectedStyle.Render("> " + opt))
		} else {
			sb.WriteString(DimmedItem.Render("  " + opt))
		}
		if i < len(s.options)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
