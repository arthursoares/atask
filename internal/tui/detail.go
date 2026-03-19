package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/atask/atask/internal/client"
)

// DetailTab identifies which tab is active in the detail pane.
type DetailTab int

const (
	TabNotes     DetailTab = iota // 0
	TabChecklist                  // 1
	TabActivity                   // 2
)

// EditNotesMsg signals the parent that the user wants to edit notes.
type EditNotesMsg struct {
	TaskID string
}

// ToggleChecklistItemMsg signals the parent to toggle a checklist item.
type ToggleChecklistItemMsg struct {
	TaskID string
	ItemID string
	Done   bool
}

// AddChecklistItemMsg signals the parent to create a new checklist item.
type AddChecklistItemMsg struct {
	TaskID string
	Title  string
}

// DeleteChecklistItemMsg signals the parent to delete a checklist item.
type DeleteChecklistItemMsg struct {
	TaskID string
	ItemID string
}

// AddCommentMsg signals the parent to add a comment activity.
type AddCommentMsg struct {
	TaskID  string
	Content string
}

// Detail is the right-hand detail pane showing task information.
type Detail struct {
	task    *client.Task
	tab     DetailTab
	height  int
	width   int
	focused bool

	// Notes tab
	notesContent string
	notesOffset  int // scroll offset in lines

	// Checklist tab
	checklist   []client.ChecklistItem
	checkCursor int
	checkOffset int

	// Activity tab
	activities     []client.Activity
	activityOffset int

	// Latest activity (always shown at bottom)
	latestActivity *client.Activity

	// Input modes
	addingComment   bool
	commentInput    textinput.Model
	addingCheckItem bool
	checkItemInput  textinput.Model
}

// NewDetail constructs a Detail with sensible defaults.
func NewDetail() Detail {
	ci := textinput.New()
	ci.Placeholder = "New item title…"

	cmt := textinput.New()
	cmt.Placeholder = "Add comment…"

	return Detail{
		checkItemInput: ci,
		commentInput:   cmt,
	}
}

// SetSize updates the pane dimensions.
func (d *Detail) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetFocused updates whether the pane has keyboard focus.
func (d *Detail) SetFocused(focused bool) {
	d.focused = focused
}

// SetTask updates the displayed task, resetting scroll offsets.
func (d *Detail) SetTask(task *client.Task) {
	d.task = task
	d.notesOffset = 0
	d.checkCursor = 0
	d.checkOffset = 0
	d.activityOffset = 0
	d.addingComment = false
	d.addingCheckItem = false
	if task != nil {
		d.notesContent = task.Notes
	} else {
		d.notesContent = ""
	}
}

// SetChecklist replaces the checklist items.
func (d *Detail) SetChecklist(items []client.ChecklistItem) {
	d.checklist = items
	if d.checkCursor >= len(items) && len(items) > 0 {
		d.checkCursor = len(items) - 1
	}
}

// SetActivities replaces the activity log and updates latestActivity.
func (d *Detail) SetActivities(activities []client.Activity) {
	d.activities = activities
	if len(activities) > 0 {
		last := activities[len(activities)-1]
		d.latestActivity = &last
	} else {
		d.latestActivity = nil
	}
}

// Update handles key presses when the detail pane is focused.
func (d Detail) Update(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	// Handle active text inputs first.
	if d.addingCheckItem {
		return d.updateCheckItemInput(msg)
	}
	if d.addingComment {
		return d.updateCommentInput(msg)
	}

	// Tab switching: 1/2/3 or Tab key.
	switch {
	case isRune(msg, '1'):
		d.tab = TabNotes
		return d, nil
	case isRune(msg, '2'):
		d.tab = TabChecklist
		return d, nil
	case isRune(msg, '3'):
		d.tab = TabActivity
		return d, nil
	case isTab(msg):
		d.tab = (d.tab + 1) % 3
		return d, nil
	}

	// Tab-specific keys.
	switch d.tab {
	case TabNotes:
		return d.updateNotes(msg)
	case TabChecklist:
		return d.updateChecklist(msg)
	case TabActivity:
		return d.updateActivity(msg)
	}

	return d, nil
}

// updateNotes handles key presses for the Notes tab.
func (d Detail) updateNotes(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	switch {
	case isRune(msg, 'j') || isDown(msg):
		d.notesOffset++
		return d, nil
	case isRune(msg, 'k') || isUp(msg):
		if d.notesOffset > 0 {
			d.notesOffset--
		}
		return d, nil
	case isRune(msg, 'e'):
		if d.task == nil {
			return d, nil
		}
		id := d.task.ID
		return d, func() tea.Msg { return EditNotesMsg{TaskID: id} }
	}
	return d, nil
}

// updateChecklist handles key presses for the Checklist tab.
func (d Detail) updateChecklist(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	n := len(d.checklist)
	switch {
	case isRune(msg, 'j') || isDown(msg):
		if d.checkCursor < n-1 {
			d.checkCursor++
			// Scroll down if cursor moves past visible window.
			if d.checkCursor >= d.checkOffset+d.visibleChecklistLines() {
				d.checkOffset = d.checkCursor - d.visibleChecklistLines() + 1
			}
		}
		return d, nil
	case isRune(msg, 'k') || isUp(msg):
		if d.checkCursor > 0 {
			d.checkCursor--
			// Scroll up if cursor moves above visible window.
			if d.checkCursor < d.checkOffset {
				d.checkOffset = d.checkCursor
			}
		}
		return d, nil
	case isRune(msg, 'x'):
		if n == 0 || d.task == nil {
			return d, nil
		}
		item := d.checklist[d.checkCursor]
		done := item.Status == 0 // toggling: 0=open → complete
		taskID := d.task.ID
		return d, func() tea.Msg {
			return ToggleChecklistItemMsg{TaskID: taskID, ItemID: item.ID, Done: done}
		}
	case isRune(msg, 'n'):
		if d.task == nil {
			return d, nil
		}
		d.addingCheckItem = true
		d.checkItemInput.SetValue("")
		d.checkItemInput.Focus()
		return d, nil
	case isRune(msg, 'd'):
		if n == 0 || d.task == nil {
			return d, nil
		}
		item := d.checklist[d.checkCursor]
		taskID := d.task.ID
		return d, func() tea.Msg {
			return DeleteChecklistItemMsg{TaskID: taskID, ItemID: item.ID}
		}
	}
	return d, nil
}

// updateActivity handles key presses for the Activity tab.
func (d Detail) updateActivity(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	switch {
	case isRune(msg, 'j') || isDown(msg):
		d.activityOffset++
		return d, nil
	case isRune(msg, 'k') || isUp(msg):
		if d.activityOffset > 0 {
			d.activityOffset--
		}
		return d, nil
	case isRune(msg, 'a'):
		if d.task == nil {
			return d, nil
		}
		d.addingComment = true
		d.commentInput.SetValue("")
		d.commentInput.Focus()
		return d, nil
	}
	return d, nil
}

// updateCheckItemInput handles key presses while typing a new checklist item.
func (d Detail) updateCheckItemInput(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	switch {
	case isEnter(msg):
		title := strings.TrimSpace(d.checkItemInput.Value())
		d.addingCheckItem = false
		d.checkItemInput.Blur()
		if title == "" || d.task == nil {
			return d, nil
		}
		taskID := d.task.ID
		return d, func() tea.Msg {
			return AddChecklistItemMsg{TaskID: taskID, Title: title}
		}
	case isEscape(msg):
		d.addingCheckItem = false
		d.checkItemInput.Blur()
		return d, nil
	default:
		var cmd tea.Cmd
		d.checkItemInput, cmd = d.checkItemInput.Update(msg)
		return d, cmd
	}
}

// updateCommentInput handles key presses while typing a new comment.
func (d Detail) updateCommentInput(msg tea.KeyPressMsg) (Detail, tea.Cmd) {
	switch {
	case isEnter(msg):
		content := strings.TrimSpace(d.commentInput.Value())
		d.addingComment = false
		d.commentInput.Blur()
		if content == "" || d.task == nil {
			return d, nil
		}
		taskID := d.task.ID
		return d, func() tea.Msg {
			return AddCommentMsg{TaskID: taskID, Content: content}
		}
	case isEscape(msg):
		d.addingComment = false
		d.commentInput.Blur()
		return d, nil
	default:
		var cmd tea.Cmd
		d.commentInput, cmd = d.commentInput.Update(msg)
		return d, cmd
	}
}

// View renders the full detail pane content.
func (d Detail) View() string {
	if d.task == nil {
		return DimmedItem.Render("Select a task to view details")
	}

	// Available inner height (no border — border is added by app.go paneView).
	// Reserve lines: title(1) + metadata(1) + blank(1) + tabbar(1) + blank(1) = 5 header lines
	// + 2 footer lines (separator + latest activity) + 1 optional input line
	const headerLines = 5
	footerLines := 2
	if d.addingComment || d.addingCheckItem {
		footerLines++
	}
	contentHeight := d.height - headerLines - footerLines
	if contentHeight < 1 {
		contentHeight = 1
	}

	var b strings.Builder

	// Title line.
	b.WriteString(DetailTitle.Render(d.task.Title))
	b.WriteString("\n")

	// Metadata line.
	b.WriteString(MetadataLine.Render(d.metadataLine()))
	b.WriteString("\n")

	b.WriteString("\n")

	// Tab bar.
	b.WriteString(d.tabBar())
	b.WriteString("\n")

	b.WriteString("\n")

	// Tab content (scrollable).
	b.WriteString(d.tabContent(contentHeight))

	b.WriteString("\n")

	// Latest activity footer.
	b.WriteString(d.latestActivityFooter())

	// Input prompt if active.
	if d.addingCheckItem {
		b.WriteString("\n")
		b.WriteString("New item: " + d.checkItemInput.View())
	} else if d.addingComment {
		b.WriteString("\n")
		b.WriteString("Comment: " + d.commentInput.View())
	}

	return b.String()
}

// metadataLine builds the metadata string with area/project, tags, and deadline.
func (d Detail) metadataLine() string {
	var parts []string
	if d.task.AreaID != nil {
		parts = append(parts, "📁 Area")
	}
	if d.task.ProjectID != nil {
		parts = append(parts, "📁 Project")
	}
	if len(d.task.Tags) > 0 {
		parts = append(parts, "🏷 "+strings.Join(d.task.Tags, ", "))
	}
	if d.task.Deadline != nil {
		parts = append(parts, "📅 "+formatDateShort(*d.task.Deadline))
	}
	if len(parts) == 0 {
		return "No metadata"
	}
	return strings.Join(parts, "  ")
}

// tabBar renders the [Notes] [Checklist N/M] [Act.] tab labels.
func (d Detail) tabBar() string {
	tabs := []string{
		d.tabLabel(TabNotes, "Notes"),
		d.tabLabel(TabChecklist, d.checklistTabLabel()),
		d.tabLabel(TabActivity, "Act."),
	}
	return strings.Join(tabs, "  ")
}

// tabLabel returns a styled tab label.
func (d Detail) tabLabel(tab DetailTab, label string) string {
	text := "[" + label + "]"
	if d.tab == tab {
		return ActiveTab.Render(text)
	}
	return InactiveTab.Render(text)
}

// checklistTabLabel returns "Checklist N/M" with completed/total counts.
func (d Detail) checklistTabLabel() string {
	if len(d.checklist) == 0 {
		return "Checklist"
	}
	completed := 0
	for _, item := range d.checklist {
		if item.Status != 0 {
			completed++
		}
	}
	return fmt.Sprintf("Checklist %d/%d", completed, len(d.checklist))
}

// tabContent returns the scrollable content for the current tab.
func (d Detail) tabContent(height int) string {
	switch d.tab {
	case TabNotes:
		return d.notesView(height)
	case TabChecklist:
		return d.checklistView(height)
	case TabActivity:
		return d.activityView(height)
	default:
		return ""
	}
}

// notesView renders the notes with scrolling.
func (d Detail) notesView(height int) string {
	if d.notesContent == "" {
		return DimmedItem.Render("No notes.")
	}
	lines := strings.Split(d.notesContent, "\n")
	start := d.notesOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// checklistView renders the checklist items with cursor.
func (d Detail) checklistView(height int) string {
	if len(d.checklist) == 0 {
		return DimmedItem.Render("No checklist items. Press n to add.")
	}

	var lines []string
	for i, item := range d.checklist {
		var checkbox string
		if item.Status != 0 {
			checkbox = "✓"
		} else {
			checkbox = "☐"
		}
		line := checkbox + " " + item.Title
		if i == d.checkCursor {
			line = SelectedItem.Render(line)
		} else if item.Status != 0 {
			line = CompletedTask.Render(line)
		}
		lines = append(lines, line)
	}

	// Apply scroll window.
	start := d.checkOffset
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// activityView renders the activity list with scrolling.
func (d Detail) activityView(height int) string {
	if len(d.activities) == 0 {
		return DimmedItem.Render("No activity yet. Press a to comment.")
	}

	var lines []string
	for _, act := range d.activities {
		lines = append(lines, d.formatActivity(act))
	}

	start := d.activityOffset
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

// formatActivity renders a single activity entry.
func (d Detail) formatActivity(act client.Activity) string {
	var actor string
	switch act.ActorType {
	case "agent":
		actor = AgentActor.Render("🤖 agent")
	default:
		actor = HumanActor.Render("👤 human")
	}
	age := timeAgo(act.CreatedAt)
	return fmt.Sprintf("%s — %s — %s (%s)", actor, act.Type, act.Content, age)
}

// latestActivityFooter renders the always-visible latest activity strip.
func (d Detail) latestActivityFooter() string {
	sep := DimmedItem.Render(strings.Repeat("─", d.width))
	if d.latestActivity == nil {
		return sep + "\n" + DimmedItem.Render("No activity yet.")
	}
	return sep + "\n" + d.formatActivity(*d.latestActivity)
}

// visibleChecklistLines returns an approximate visible height for the checklist.
func (d Detail) visibleChecklistLines() int {
	const headerLines = 5
	const footerLines = 2
	h := d.height - headerLines - footerLines
	if h < 1 {
		return 1
	}
	return h
}

// formatDateShort converts an ISO date string to a short human-readable form.
func formatDateShort(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		// Try date-only format.
		t, err = time.Parse("2006-01-02", s)
		if err != nil {
			return s
		}
	}
	return t.Format("Jan 2")
}

// timeAgo returns a short human-readable "time since" string.
func timeAgo(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
