package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// ---------------------------------------------------------------------------
// Open functions
// ---------------------------------------------------------------------------

// openPalette builds the command list based on current context and opens the
// palette overlay.
func (m *Model) openPalette() {
	ti := textinput.New()
	ti.Placeholder = "type a command…"
	ti.Focus()

	cmds := []PaletteCommand{
		// Navigation
		{Name: "Go to Inbox", Category: "navigation", Action: func() tea.Cmd {
			m.currentView = "inbox"
			return m.cmdLoadInbox()
		}},
		{Name: "Go to Today", Category: "navigation", Action: func() tea.Cmd {
			m.currentView = "today"
			return m.cmdLoadToday()
		}},
		{Name: "Go to Upcoming", Category: "navigation", Action: func() tea.Cmd {
			m.currentView = "upcoming"
			return m.cmdLoadUpcoming()
		}},
		{Name: "Go to Someday", Category: "navigation", Action: func() tea.Cmd {
			m.currentView = "someday"
			return m.cmdLoadSomeday()
		}},
		{Name: "Go to Logbook", Category: "navigation", Action: func() tea.Cmd {
			m.currentView = "logbook"
			return m.cmdLoadLogbook()
		}},
		// System
		{Name: "Refresh", Category: "system", Action: func() tea.Cmd {
			return m.refreshCurrentView()
		}},
		{Name: "Help", Category: "system", Action: func() tea.Cmd {
			m.showHelp = true
			return nil
		}},
		{Name: "Quit", Category: "system", Action: func() tea.Cmd {
			return tea.Quit
		}},
	}

	// Task operations — only available when a task is selected.
	if m.selectedTask != nil {
		task := m.selectedTask
		cmds = append(cmds,
			PaletteCommand{Name: "Complete Task", Category: "task", Action: func() tea.Cmd {
				return m.cmdCompleteTask(task.ID)
			}},
			PaletteCommand{Name: "Cancel Task", Category: "task", Action: func() tea.Cmd {
				return m.cmdCancelTask(task.ID)
			}},
			PaletteCommand{Name: "Delete Task", Category: "task", Action: func() tea.Cmd {
				m.openConfirm(fmt.Sprintf("Delete task %q?", task.Title), func() tea.Cmd {
					return m.cmdDeleteTask(task.ID)
				})
				return nil
			}},
			PaletteCommand{Name: "Schedule Task", Category: "task", Action: func() tea.Cmd {
				m.openSchedule(task.ID)
				return nil
			}},
			PaletteCommand{Name: "Move to Project", Category: "task", Action: func() tea.Cmd {
				items := make([]PickerItem, 0, len(m.projects))
				items = append(items, PickerItem{ID: "", Label: "No project (Inbox)"})
				for _, p := range m.projects {
					items = append(items, PickerItem{ID: p.ID, Label: p.Name})
				}
				m.openPicker("Move to Project", items, func(id string) tea.Cmd {
					if id == "" {
						return m.cmdMoveToProject(task.ID, nil)
					}
					return m.cmdMoveToProject(task.ID, &id)
				})
				return nil
			}},
		)
	}

	m.palette = &PaletteState{
		Input:    ti,
		Commands: cmds,
		Filtered: cmds,
		Cursor:   0,
	}
}

// openSearch creates a SearchState with a focused textinput.
func (m *Model) openSearch() {
	ti := textinput.New()
	ti.Placeholder = "search tasks…"
	ti.Focus()
	m.search = &SearchState{Input: ti}
}

// openConfirm opens a confirmation dialog with the given message and callback.
func (m *Model) openConfirm(message string, onYes func() tea.Cmd) {
	m.confirm = &ConfirmState{
		Message: message,
		OnYes:   onYes,
	}
}

// openPicker opens a generic picker overlay.
func (m *Model) openPicker(title string, items []PickerItem, onSelect func(string) tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "filter…"
	ti.Focus()
	m.picker = &PickerState{
		Title:    title,
		Input:    ti,
		Items:    items,
		Filtered: items,
		Cursor:   0,
		OnSelect: func(item PickerItem) tea.Cmd {
			return onSelect(item.ID)
		},
	}
}

// openSchedule opens the schedule picker for the given task.
func (m *Model) openSchedule(taskID string) {
	m.schedule = &ScheduleState{
		TaskID:  taskID,
		Options: []string{"Inbox", "Today", "Someday"},
		Cursor:  0,
	}
}

// openInput opens a generic single-line text input prompt.
func (m *Model) openInput(prompt string, onDone func(string) tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	m.inputPrompt = &InputState{
		Prompt: prompt,
		Input:  ti,
		OnDone: onDone,
	}
}

// ---------------------------------------------------------------------------
// Update functions
// ---------------------------------------------------------------------------

// updatePalette handles key events for the command palette overlay.
func (m Model) updatePalette(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.palette
	if p == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape):
		m.palette = nil
		return m, nil

	case key.Matches(msg, Keys.Enter):
		if len(p.Filtered) == 0 {
			return m, nil
		}
		cmd := p.Filtered[p.Cursor]
		m.palette = nil
		return m, cmd.Action()

	case key.Matches(msg, Keys.Up):
		if p.Cursor > 0 {
			p.Cursor--
		}
		m.palette = p
		return m, nil

	case key.Matches(msg, Keys.Down):
		if p.Cursor < len(p.Filtered)-1 {
			p.Cursor++
		}
		m.palette = p
		return m, nil

	default:
		var cmd tea.Cmd
		p.Input, cmd = p.Input.Update(msg)
		query := p.Input.Value()
		p.Filtered = filterCommands(p.Commands, query)
		if p.Cursor >= len(p.Filtered) {
			p.Cursor = max(0, len(p.Filtered)-1)
		}
		m.palette = p
		return m, cmd
	}
}

// updateSearch handles key events for the search bar overlay.
func (m Model) updateSearch(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := m.search
	if s == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape):
		m.search = nil
		// Reload the current view to restore the unfiltered task list.
		return m, m.refreshCurrentView()

	case key.Matches(msg, Keys.Enter):
		// Commit search — leave overlay open but unfocus.
		m.search = s
		return m, nil

	default:
		var cmd tea.Cmd
		s.Input, cmd = s.Input.Update(msg)
		m.search = s
		return m, cmd
	}
}

// updateConfirm handles key events for the confirmation dialog overlay.
func (m Model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := m.confirm
	if c == nil {
		return m, nil
	}

	switch {
	case msg.Code == 'y' || msg.Code == 'Y':
		onYes := c.OnYes
		m.confirm = nil
		return m, onYes()

	case msg.Code == 'n' || msg.Code == 'N' || key.Matches(msg, Keys.Escape):
		m.confirm = nil
		return m, nil
	}

	return m, nil
}

// updatePicker handles key events for the generic picker overlay.
func (m Model) updatePicker(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.picker
	if p == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape):
		m.picker = nil
		return m, nil

	case key.Matches(msg, Keys.Enter):
		if len(p.Filtered) == 0 {
			return m, nil
		}
		selected := p.Filtered[p.Cursor]
		onSelect := p.OnSelect
		m.picker = nil
		return m, onSelect(selected)

	case key.Matches(msg, Keys.Up):
		if p.Cursor > 0 {
			p.Cursor--
		}
		m.picker = p
		return m, nil

	case key.Matches(msg, Keys.Down):
		if p.Cursor < len(p.Filtered)-1 {
			p.Cursor++
		}
		m.picker = p
		return m, nil

	default:
		var cmd tea.Cmd
		p.Input, cmd = p.Input.Update(msg)
		query := p.Input.Value()
		p.Filtered = filterItems(p.Items, query)
		if p.Cursor >= len(p.Filtered) {
			p.Cursor = max(0, len(p.Filtered)-1)
		}
		m.picker = p
		return m, cmd
	}
}

// updateSchedule handles key events for the schedule picker overlay.
func (m Model) updateSchedule(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := m.schedule
	if s == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape):
		m.schedule = nil
		return m, nil

	case key.Matches(msg, Keys.Enter):
		taskID := s.TaskID
		opt := strings.ToLower(s.Options[s.Cursor])
		m.schedule = nil
		return m, m.cmdUpdateSchedule(taskID, opt)

	case key.Matches(msg, Keys.Up):
		if s.Cursor > 0 {
			s.Cursor--
		}
		m.schedule = s
		return m, nil

	case key.Matches(msg, Keys.Down):
		if s.Cursor < len(s.Options)-1 {
			s.Cursor++
		}
		m.schedule = s
		return m, nil
	}

	return m, nil
}

// updateInput handles key events for the generic input prompt overlay.
func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	ip := m.inputPrompt
	if ip == nil {
		return m, nil
	}

	switch {
	case key.Matches(msg, Keys.Escape):
		m.inputPrompt = nil
		return m, nil

	case key.Matches(msg, Keys.Enter):
		value := ip.Input.Value()
		onDone := ip.OnDone
		m.inputPrompt = nil
		return m, onDone(value)

	default:
		var cmd tea.Cmd
		ip.Input, cmd = ip.Input.Update(msg)
		m.inputPrompt = ip
		return m, cmd
	}
}

// ---------------------------------------------------------------------------
// Render functions
// ---------------------------------------------------------------------------

// renderPaletteOverlay renders the command palette as an overlay string.
func (m Model) renderPaletteOverlay() string {
	p := m.palette
	if p == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(p.Input.View() + "\n\n")

	maxVisible := 10
	start := 0
	if p.Cursor >= maxVisible {
		start = p.Cursor - maxVisible + 1
	}

	shown := p.Filtered
	if start > 0 {
		shown = shown[start:]
	}
	if len(shown) > maxVisible {
		shown = shown[:maxVisible]
	}

	for i, cmd := range shown {
		actualIdx := start + i
		prefix := MutedStyle.Render(cmd.Category+": ")
		line := prefix + cmd.Name
		if actualIdx == p.Cursor {
			line = SelectedStyle.Render(fmt.Sprintf(" ▸ %-40s", cmd.Category+": "+cmd.Name))
		} else {
			line = "   " + prefix + cmd.Name
		}
		b.WriteString(line + "\n")
	}

	if len(p.Filtered) == 0 {
		b.WriteString(MutedStyle.Render("  no matches") + "\n")
	}

	overlayWidth := 52
	return OverlayBorder.Width(overlayWidth).Render(b.String())
}

// renderSearchOverlay renders the search bar as a small overlay string.
func (m Model) renderSearchOverlay() string {
	s := m.search
	if s == nil {
		return ""
	}
	bar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorSecondary)).
		Padding(0, 1).
		Render("/ " + s.Input.View())
	return bar
}

// renderHelpOverlay renders the keybinding help overlay.
func (m Model) renderHelpOverlay() string {
	if !m.showHelp {
		return ""
	}

	lines := []string{
		TitleStyle.Render("Keyboard Shortcuts"),
		"",
		MutedStyle.Render("Navigation"),
		"  " + Keys.Up.Help().Key + "  " + Keys.Up.Help().Desc,
		"  " + Keys.Down.Help().Key + "  " + Keys.Down.Help().Desc,
		"  " + Keys.Tab.Help().Key + "  " + Keys.Tab.Help().Desc,
		"  " + Keys.ShiftTab.Help().Key + "  " + Keys.ShiftTab.Help().Desc,
		"",
		MutedStyle.Render("Tasks"),
		"  " + Keys.New.Help().Key + "  " + Keys.New.Help().Desc,
		"  " + Keys.Edit.Help().Key + "  " + Keys.Edit.Help().Desc,
		"  " + Keys.Complete.Help().Key + "  " + Keys.Complete.Help().Desc,
		"  " + Keys.Cancel.Help().Key + "  " + Keys.Cancel.Help().Desc,
		"  " + Keys.Delete.Help().Key + "  " + Keys.Delete.Help().Desc,
		"  " + Keys.Schedule.Help().Key + "  " + Keys.Schedule.Help().Desc,
		"  " + Keys.Move.Help().Key + "  " + Keys.Move.Help().Desc,
		"  " + Keys.Tag.Help().Key + "  " + Keys.Tag.Help().Desc,
		"  " + Keys.Location.Help().Key + "  " + Keys.Location.Help().Desc,
		"  " + Keys.Comment.Help().Key + "  " + Keys.Comment.Help().Desc,
		"",
		MutedStyle.Render("Global"),
		"  " + Keys.Palette.Help().Key + "  " + Keys.Palette.Help().Desc,
		"  " + Keys.Search.Help().Key + "  " + Keys.Search.Help().Desc,
		"  " + Keys.Refresh.Help().Key + "  " + Keys.Refresh.Help().Desc,
		"  " + Keys.Quit.Help().Key + "  " + Keys.Quit.Help().Desc,
		"",
		MutedStyle.Render("[Esc] close"),
	}

	content := strings.Join(lines, "\n")
	return OverlayBorder.Width(44).Render(content)
}

// renderConfirmOverlay renders the confirmation dialog as an overlay string.
func (m Model) renderConfirmOverlay() string {
	c := m.confirm
	if c == nil {
		return ""
	}
	body := c.Message + "\n\n" + MutedStyle.Render("[y] yes  [n/Esc] no")
	return OverlayBorder.Width(50).Render(body)
}

// renderPickerOverlay renders the generic picker overlay.
func (m Model) renderPickerOverlay() string {
	p := m.picker
	if p == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render(p.Title) + "\n\n")
	b.WriteString(p.Input.View() + "\n\n")

	maxVisible := 8
	start := 0
	if p.Cursor >= maxVisible {
		start = p.Cursor - maxVisible + 1
	}

	shown := p.Filtered
	if start > 0 {
		shown = shown[start:]
	}
	if len(shown) > maxVisible {
		shown = shown[:maxVisible]
	}

	for i, item := range shown {
		actualIdx := start + i
		if actualIdx == p.Cursor {
			b.WriteString(SelectedStyle.Render(fmt.Sprintf(" ▸ %-36s", item.Label)) + "\n")
		} else {
			b.WriteString("   " + item.Label + "\n")
		}
	}

	if len(p.Filtered) == 0 {
		b.WriteString(MutedStyle.Render("  no matches") + "\n")
	}

	return OverlayBorder.Width(44).Render(b.String())
}

// renderScheduleOverlay renders the schedule picker overlay.
func (m Model) renderScheduleOverlay() string {
	s := m.schedule
	if s == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(TitleStyle.Render("Schedule") + "\n\n")

	for i, opt := range s.Options {
		if i == s.Cursor {
			b.WriteString(SelectedStyle.Render(fmt.Sprintf(" ▸ %-20s", opt)) + "\n")
		} else {
			b.WriteString("   " + opt + "\n")
		}
	}

	b.WriteString("\n" + MutedStyle.Render("[j/k] navigate  [Enter] select  [Esc] cancel"))

	return OverlayBorder.Width(36).Render(b.String())
}

// renderInputOverlay renders the generic input prompt overlay.
func (m Model) renderInputOverlay() string {
	ip := m.inputPrompt
	if ip == nil {
		return ""
	}

	body := ip.Prompt + "\n\n" + ip.Input.View() + "\n\n" + MutedStyle.Render("[Enter] confirm  [Esc] cancel")
	return OverlayBorder.Width(50).Render(body)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// fuzzyMatch returns true if every character of input appears in target in
// order (case-insensitive subsequence match).
func fuzzyMatch(input, target string) bool {
	input = strings.ToLower(input)
	target = strings.ToLower(target)
	i := 0
	for _, r := range target {
		if i < len(input) && rune(input[i]) == r {
			i++
		}
	}
	return i == len(input)
}

// filterCommands returns palette commands whose name fuzzy-matches query.
// An empty query returns all commands.
func filterCommands(cmds []PaletteCommand, query string) []PaletteCommand {
	if query == "" {
		return cmds
	}
	out := make([]PaletteCommand, 0, len(cmds))
	for _, c := range cmds {
		if fuzzyMatch(query, c.Name) || fuzzyMatch(query, c.Category) {
			out = append(out, c)
		}
	}
	return out
}

// filterItems returns picker items whose label fuzzy-matches query.
// An empty query returns all items.
func filterItems(items []PickerItem, query string) []PickerItem {
	if query == "" {
		return items
	}
	out := make([]PickerItem, 0, len(items))
	for _, item := range items {
		if fuzzyMatch(query, item.Label) {
			out = append(out, item)
		}
	}
	return out
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
