package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atask/atask/internal/client"
)

// List is the central task-list pane model.
type List struct {
	tasks    []client.Task
	sections []client.Section
	cursor   int
	height   int
	width    int
	offset   int
	focused  bool
	title    string

	editing   bool
	editInput textinput.Model

	filter   string
	filtered []int // indices into tasks matching filter
}

// NewList constructs a List with sensible defaults.
func NewList() List {
	ti := textinput.New()
	ti.Placeholder = "Task title…"

	return List{
		editInput: ti,
		title:     "Inbox",
	}
}

// SetSize updates the pane dimensions.
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// SetFocused marks whether the list has keyboard focus.
func (l *List) SetFocused(focused bool) {
	l.focused = focused
}

// SetTasks replaces the task slice and resets cursor/scroll.
func (l *List) SetTasks(tasks []client.Task, title string) {
	l.tasks = tasks
	l.title = title
	l.cursor = 0
	l.offset = 0
	l.editing = false
	l.rebuildFilter()
}

// SetSections sets the section list (used for grouping project tasks).
func (l *List) SetSections(sections []client.Section) {
	l.sections = sections
}

// SetFilter updates the search filter and rebuilds the filtered index.
func (l *List) SetFilter(filter string) {
	l.filter = filter
	l.cursor = 0
	l.offset = 0
	l.rebuildFilter()
}

// rebuildFilter rebuilds the filtered index from current tasks and filter string.
func (l *List) rebuildFilter() {
	lower := strings.ToLower(l.filter)
	l.filtered = nil
	for i, t := range l.tasks {
		if lower == "" || strings.Contains(strings.ToLower(t.Title), lower) {
			l.filtered = append(l.filtered, i)
		}
	}
}

// visibleCount returns the number of tasks visible after filtering.
func (l List) visibleCount() int {
	return len(l.filtered)
}

// taskAt returns the task at the given filtered-list position (or nil).
func (l List) taskAt(pos int) *client.Task {
	if pos < 0 || pos >= len(l.filtered) {
		return nil
	}
	idx := l.filtered[pos]
	if idx < 0 || idx >= len(l.tasks) {
		return nil
	}
	return &l.tasks[idx]
}

// SelectedTask returns a pointer to the currently highlighted task, or nil.
func (l List) SelectedTask() *client.Task {
	return l.taskAt(l.cursor)
}

// Update handles key presses when the list has focus.
func (l List) Update(msg tea.KeyPressMsg) (List, tea.Cmd) {
	// In editing mode, route all keys to the textinput.
	if l.editing {
		return l.updateEditInput(msg)
	}

	switch {
	// Movement
	case isRune(msg, 'j') || isDown(msg):
		l.moveCursor(1)

	case isRune(msg, 'k') || isUp(msg):
		l.moveCursor(-1)

	case isRune(msg, 'g'):
		l.cursor = 0
		l.adjustOffset()

	case isRune(msg, 'G'):
		if l.visibleCount() > 0 {
			l.cursor = l.visibleCount() - 1
		}
		l.adjustOffset()

	// Selection
	case isEnter(msg):
		if t := l.SelectedTask(); t != nil {
			id := t.ID
			return l, func() tea.Msg { return TaskSelectedMsg{ID: id} }
		}

	// Actions signalled to parent
	case isRune(msg, 'n'):
		return l, func() tea.Msg { return CreateTaskSignal{} }

	case isRune(msg, 'x'):
		if t := l.SelectedTask(); t != nil {
			id := t.ID
			return l, func() tea.Msg { return TaskCompletedMsg{ID: id} }
		}

	case isRune(msg, 'X'):
		return l, func() tea.Msg { return CancelTaskSignal{} }

	case isRune(msg, 'd'):
		return l, func() tea.Msg { return DeleteTaskSignal{} }

	case isRune(msg, 's'):
		return l, func() tea.Msg { return ScheduleTaskSignal{} }

	case isRune(msg, 'm'):
		return l, func() tea.Msg { return MoveTaskSignal{} }

	// Inline edit
	case isRune(msg, 'e'):
		if t := l.SelectedTask(); t != nil {
			l.editing = true
			l.editInput.SetValue(t.Title)
			l.editInput.Focus()
		}
	}

	return l, nil
}

// updateEditInput routes key events to the textinput when editing.
func (l List) updateEditInput(msg tea.KeyPressMsg) (List, tea.Cmd) {
	switch {
	case isEscape(msg):
		l.editing = false
		l.editInput.Blur()
		return l, nil

	case isEnter(msg):
		newTitle := strings.TrimSpace(l.editInput.Value())
		l.editing = false
		l.editInput.Blur()
		if newTitle != "" {
			if t := l.SelectedTask(); t != nil {
				id := t.ID
				return l, func() tea.Msg { return TitleUpdatedMsg{ID: id, Title: newTitle} }
			}
		}
		return l, nil
	}

	// Forward all other keys to the textinput model.
	var cmd tea.Cmd
	l.editInput, cmd = l.editInput.Update(msg)
	return l, cmd
}

// moveCursor moves the cursor by delta and adjusts the scroll offset.
func (l *List) moveCursor(delta int) {
	n := l.visibleCount()
	if n == 0 {
		return
	}
	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if l.cursor >= n {
		l.cursor = n - 1
	}
	l.adjustOffset()
}

// adjustOffset ensures the cursor row is visible within the viewport.
func (l *List) adjustOffset() {
	viewHeight := l.listHeight()
	if viewHeight <= 0 {
		return
	}
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+viewHeight {
		l.offset = l.cursor - viewHeight + 1
	}
}

// listHeight returns the number of task rows visible (total height minus title bar).
func (l List) listHeight() int {
	h := l.height - 1 // subtract title bar row
	if h < 0 {
		return 0
	}
	return h
}

// View renders the list pane as a string.
func (l List) View() string {
	if l.height <= 0 {
		return ""
	}

	innerWidth := l.width
	if innerWidth < 1 {
		innerWidth = 1
	}

	var lines []string

	// Title bar.
	n := l.visibleCount()
	taskWord := "tasks"
	if n == 1 {
		taskWord = "task"
	}
	titleText := fmt.Sprintf("%s — %d %s", l.title, n, taskWord)
	titleLine := lipgloss.NewStyle().Bold(true).Width(innerWidth).Render(titleText)
	lines = append(lines, titleLine)

	// Task rows.
	viewHeight := l.listHeight()
	for i := 0; i < viewHeight; i++ {
		pos := l.offset + i
		if pos >= n {
			// Blank padding line.
			lines = append(lines, strings.Repeat(" ", innerWidth))
			continue
		}
		t := l.taskAt(pos)
		if t == nil {
			lines = append(lines, strings.Repeat(" ", innerWidth))
			continue
		}

		selected := pos == l.cursor
		var row string
		if selected && l.editing {
			row = l.renderEditRow(t, innerWidth)
		} else {
			row = l.renderTaskRow(t, selected, innerWidth)
		}
		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}

// renderTaskRow renders a single task row.
func (l List) renderTaskRow(t *client.Task, selected bool, width int) string {
	// Status icon.
	var statusIcon string
	switch t.Status {
	case 1: // completed
		statusIcon = "✓"
	case 2: // cancelled
		statusIcon = "✗"
	default: // open
		statusIcon = "☐"
	}

	// Right side annotation: deadline or project name or location.
	rightText := l.rightAnnotation(t)

	// Title styling.
	title := t.Title
	isDone := t.Status == 1 || t.Status == 2

	// Compute available width for the title:
	// "☐ " (2 chars) + title + padding + right annotation
	rightWidth := lipgloss.Width(rightText)
	iconWidth := lipgloss.Width(statusIcon) + 1 // icon + space
	// Minimum gap between title and right text.
	minGap := 1
	maxTitleWidth := width - iconWidth - minGap - rightWidth
	if maxTitleWidth < 1 {
		maxTitleWidth = 1
	}

	// Truncate title if needed.
	titleRunes := []rune(title)
	if len(titleRunes) > maxTitleWidth {
		titleRunes = titleRunes[:maxTitleWidth-1]
		title = string(titleRunes) + "…"
	} else {
		title = string(titleRunes)
	}

	// Pad to fill the gap.
	usedWidth := iconWidth + lipgloss.Width(title) + rightWidth
	padLen := width - usedWidth
	if padLen < 1 {
		padLen = 1
	}
	pad := strings.Repeat(" ", padLen)

	line := statusIcon + " " + title + pad + rightText

	// Apply styling.
	if isDone {
		// Dim the whole row; title gets strikethrough via CompletedTask style.
		doneTitle := CompletedTask.Render(statusIcon + " " + title)
		doneRight := DimmedItem.Render(pad + rightText)
		line = doneTitle + doneRight
		if selected {
			line = SelectedTask.Width(width).Render(line)
		}
	} else if selected {
		line = SelectedTask.Width(width).Render(line)
	} else {
		line = lipgloss.NewStyle().Width(width).Render(line)
	}

	return line
}

// renderEditRow renders the inline edit textinput for the selected task.
func (l List) renderEditRow(t *client.Task, width int) string {
	// Show "☐ " prefix + text input.
	prefix := "☐ "
	prefixWidth := lipgloss.Width(prefix)
	inputWidth := width - prefixWidth
	if inputWidth < 4 {
		inputWidth = 4
	}
	return SelectedTask.Width(width).Render(prefix + l.editInput.View())
}

// rightAnnotation returns the short right-side label for a task row.
func (l List) rightAnnotation(t *client.Task) string {
	// Deadline takes priority.
	if t.Deadline != nil && *t.Deadline != "" {
		return l.formatDeadline(*t.Deadline)
	}
	// Project name (look up section title for project tasks).
	if t.ProjectID != nil {
		for _, sec := range l.sections {
			if t.SectionID != nil && sec.ID == *t.SectionID {
				return "📁 " + sec.Title
			}
		}
		// Fallback: just a folder icon (project name not available without extra lookup).
		return "📁"
	}
	// Location.
	if t.LocationID != nil {
		return "📍"
	}
	return ""
}

// formatDeadline formats a deadline string (YYYY-MM-DD) as a short label,
// applying red styling if the deadline is in the past.
func (l List) formatDeadline(deadline string) string {
	t, err := time.Parse("2006-01-02", deadline)
	if err != nil {
		// Try RFC3339 format as fallback.
		t, err = time.Parse(time.RFC3339, deadline)
		if err != nil {
			return deadline
		}
	}

	label := t.Format("Jan 2")
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if t.Before(today) {
		return OverdueDeadline.Render(label)
	}
	return label
}

// --- Signal message types (signals to parent App, not persisted) ---

// CreateTaskSignal signals the parent to open task creation UI.
type CreateTaskSignal struct{}

// CancelTaskSignal signals the parent to cancel (mark cancelled) the selected task.
type CancelTaskSignal struct{}

// DeleteTaskSignal signals the parent to delete the selected task.
type DeleteTaskSignal struct{}

// ScheduleTaskSignal signals the parent to open the schedule picker.
type ScheduleTaskSignal struct{}

// MoveTaskSignal signals the parent to open the move-to-project picker.
type MoveTaskSignal struct{}
