package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
)

// View name constants used throughout the TUI.
const (
	viewInbox    = "inbox"
	viewToday    = "today"
	viewUpcoming = "upcoming"
	viewSomeday  = "someday"
	viewLogbook  = "logbook"
	viewProject  = "project"
	viewArea     = "area"
)

// Update is the bubbletea update function — the single entry point for all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Window resize ──────────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// ── Key input ─────────────────────────────────────────────────────────────
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	// ── Data loaded ───────────────────────────────────────────────────────────
	case TasksLoadedMsg:
		m.tasks = msg.Tasks
		m.listCursor = 0
		m.listScroll = 0
		m.selectFirstTask()
		return m, nil

	case AreasLoadedMsg:
		m.areas = msg.Areas
		m.rebuildSidebar()
		return m, nil

	case ProjectsLoadedMsg:
		m.projects = msg.Projects
		m.rebuildSidebar()
		return m, nil

	case TagsLoadedMsg:
		m.tags = msg.Tags
		m.rebuildSidebar()
		return m, nil

	case LocationsLoadedMsg:
		m.locations = msg.Locations
		return m, nil

	case ChecklistLoadedMsg:
		m.checklist = msg.Items
		m.checkCursor = 0
		return m, nil

	case ActivitiesLoadedMsg:
		m.activities = msg.Activities
		return m, nil

	// ── Mutation results ───────────────────────────────────────────────────────
	case RefreshMsg:
		return m, m.refreshCurrentView()

	case DetailRefreshMsg:
		if m.selectedTask != nil && m.selectedTask.ID == msg.TaskID {
			return m, m.refreshDetail()
		}
		return m, nil

	case TaskCreatedMsg:
		return m, m.refreshCurrentView()

	case FlashMsg:
		m.statusFlash = msg.Message
		return m, nil

	case ClearFlashMsg:
		m.statusFlash = ""
		return m, nil

	// ── SSE lifecycle ─────────────────────────────────────────────────────────
	case SSEStartedMsg:
		m.sseEvents = msg.Events
		return m, cmdListenSSE(m.sseEvents)

	case SSEEventMsg:
		cmd := m.handleSSEEvent(msg.Event)
		return m, tea.Batch(cmd, cmdListenSSE(m.sseEvents))

	case SSEDisconnectedMsg:
		m.sseEvents = nil
		m.statusErr = "SSE disconnected"
		return m, nil

	// ── Error ─────────────────────────────────────────────────────────────────
	case ErrorMsg:
		m.statusErr = msg.Err.Error()
		return m, nil
	}

	return m, nil
}

// handleKey is the keyboard dispatcher. Overlay intercept runs first, then
// global bindings, then pane-specific handlers.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 1. Command palette
	if m.palette != nil {
		return m.updatePalette(msg)
	}

	// 2. Search overlay
	if m.search != nil {
		return m.updateSearch(msg)
	}

	// 3. Help screen — any key closes it
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	// 4. Confirm dialog
	if m.confirm != nil {
		return m.updateConfirm(msg)
	}

	// 5. Picker overlay
	if m.picker != nil {
		return m.updatePicker(msg)
	}

	// 6. Schedule overlay
	if m.schedule != nil {
		return m.updateSchedule(msg)
	}

	// 7. Input prompt overlay
	if m.inputPrompt != nil {
		return m.updateInput(msg)
	}

	// 8. Inline task-title editing
	if m.editing {
		return m.updateEditing(msg)
	}

	// 9. Global keys (available from any pane)
	switch {
	case key.Matches(msg, Keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, Keys.Tab):
		m.cycleFocus(+1)
		return m, nil

	case key.Matches(msg, Keys.ShiftTab):
		m.cycleFocus(-1)
		return m, nil

	case key.Matches(msg, Keys.Palette):
		m.openPalette()
		return m, nil

	case key.Matches(msg, Keys.Search):
		m.openSearch()
		return m, nil

	case key.Matches(msg, Keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, Keys.Refresh):
		return m, m.refreshCurrentView()
	}

	// 10. Pane-specific keys
	switch m.focusedPane {
	case SidebarPane:
		return m.updateSidebar(msg)
	case ListPane:
		return m.updateList(msg)
	case DetailPane:
		return m.updateDetail(msg)
	}

	return m, nil
}

// ── Focus cycling ──────────────────────────────────────────────────────────────

// cycleFocus moves focus one step forward (dir=+1) or backward (dir=-1)
// through the three panes, wrapping around.
func (m *Model) cycleFocus(dir int) {
	m.focusedPane = Pane((int(m.focusedPane) + dir + NumPanes) % NumPanes)
}

// ── Task selection helper ──────────────────────────────────────────────────────

// selectFirstTask sets selectedTask to the first element of m.tasks, or nil.
func (m *Model) selectFirstTask() {
	if len(m.tasks) > 0 {
		t := m.tasks[0]
		m.selectedTask = &t
	} else {
		m.selectedTask = nil
	}
}

// ── Sidebar rebuild ────────────────────────────────────────────────────────────

// rebuildSidebar reconstructs m.sidebarItems from current areas/projects/tags.
func (m *Model) rebuildSidebar() {
	items := []SidebarItem{}

	// Fixed views — Title-case the view ID for display.
	for _, v := range []string{viewInbox, viewToday, viewUpcoming, viewSomeday, viewLogbook} {
		icon := ViewIcons[v]
		label := icon + " " + strings.ToUpper(v[:1]) + v[1:]
		items = append(items, SidebarItem{
			Label: label,
			Kind:  "view",
			ID:    v,
		})
	}

	// Areas header
	items = append(items, SidebarItem{Label: "AREAS", Kind: "header"})

	for _, a := range m.areas {
		expanded := m.areaExpanded[a.ID]
		items = append(items, SidebarItem{
			Label:    a.Title,
			Kind:     "area",
			ID:       a.ID,
			Expanded: expanded,
		})
		if expanded {
			for _, p := range m.projects {
				if p.AreaID != nil && *p.AreaID == a.ID {
					items = append(items, SidebarItem{
						Label:  p.Title,
						Kind:   "project",
						ID:     p.ID,
						Indent: 1,
					})
				}
			}
		}
	}

	// Tags header
	items = append(items, SidebarItem{Label: "TAGS", Kind: "header"})

	for _, t := range m.tags {
		items = append(items, SidebarItem{
			Label: t.Title,
			Kind:  "tag",
			ID:    t.ID,
		})
	}

	m.sidebarItems = items
}

// ── Sidebar pane keys ──────────────────────────────────────────────────────────

func (m Model) updateSidebar(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		m.sidebarCursor--
		// Skip header items
		for m.sidebarCursor >= 0 && m.sidebarItems[m.sidebarCursor].Kind == "header" {
			m.sidebarCursor--
		}
		if m.sidebarCursor < 0 {
			// Clamp to first selectable item
			m.sidebarCursor = 0
			for m.sidebarCursor < len(m.sidebarItems) && m.sidebarItems[m.sidebarCursor].Kind == "header" {
				m.sidebarCursor++
			}
		}
		return m, nil

	case key.Matches(msg, Keys.Down):
		m.sidebarCursor++
		// Skip header items
		for m.sidebarCursor < len(m.sidebarItems) && m.sidebarItems[m.sidebarCursor].Kind == "header" {
			m.sidebarCursor++
		}
		if m.sidebarCursor >= len(m.sidebarItems) {
			// Clamp to last selectable item
			m.sidebarCursor = len(m.sidebarItems) - 1
			for m.sidebarCursor > 0 && m.sidebarItems[m.sidebarCursor].Kind == "header" {
				m.sidebarCursor--
			}
		}
		return m, nil

	case key.Matches(msg, Keys.Enter):
		if m.sidebarCursor < 0 || m.sidebarCursor >= len(m.sidebarItems) {
			return m, nil
		}
		item := m.sidebarItems[m.sidebarCursor]
		switch item.Kind {
		case "view":
			m.currentView = item.ID
			m.statusContext = item.Label
			m.listCursor = 0
			m.listScroll = 0
			return m, m.refreshCurrentView()

		case "area":
			m.areaExpanded[item.ID] = !m.areaExpanded[item.ID]
			m.rebuildSidebar()
			return m, nil

		case "project":
			m.currentView = viewProject + ":" + item.ID
			m.statusContext = item.Label
			m.listCursor = 0
			m.listScroll = 0
			return m, m.refreshCurrentView()

		case "tag":
			m.statusFlash = "tag filter coming soon"
			return m, nil
		}

	case key.Matches(msg, Keys.New):
		m.openInput("New area or project name:", func(name string) tea.Cmd {
			if name == "" {
				return nil
			}
			return func() tea.Msg { return FlashMsg{Message: "area/project creation coming soon"} }
		})
		return m, nil

	case key.Matches(msg, Keys.Edit):
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sidebarItems) {
			item := m.sidebarItems[m.sidebarCursor]
			if item.Kind == "area" || item.Kind == "project" || item.Kind == "tag" {
				m.openInput("Rename to:", func(name string) tea.Cmd {
					if name == "" {
						return nil
					}
					return func() tea.Msg { return FlashMsg{Message: "rename coming soon"} }
				})
			}
		}
		return m, nil

	case key.Matches(msg, Keys.Delete):
		if m.sidebarCursor >= 0 && m.sidebarCursor < len(m.sidebarItems) {
			item := m.sidebarItems[m.sidebarCursor]
			if item.Kind == "area" || item.Kind == "project" || item.Kind == "tag" {
				m.openConfirm("Delete "+item.Label+"?", func() tea.Cmd {
					return func() tea.Msg { return FlashMsg{Message: "delete coming soon"} }
				})
			}
		}
		return m, nil
	}

	return m, nil
}

// ── List pane keys ─────────────────────────────────────────────────────────────

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.listCursor > 0 {
			m.listCursor--
			t := m.tasks[m.listCursor]
			m.selectedTask = &t
		}
		return m, nil

	case key.Matches(msg, Keys.Down):
		if m.listCursor < len(m.tasks)-1 {
			m.listCursor++
			t := m.tasks[m.listCursor]
			m.selectedTask = &t
		}
		return m, nil

	case key.Matches(msg, Keys.Enter):
		if m.selectedTask != nil {
			m.focusedPane = DetailPane
		}
		return m, nil

	case key.Matches(msg, Keys.New):
		m.openInput("Task title:", func(title string) tea.Cmd {
			if title == "" {
				return nil
			}
			return m.cmdCreateTask(title)
		})
		return m, nil

	case key.Matches(msg, Keys.Complete):
		if m.selectedTask != nil {
			id := m.selectedTask.ID
			return m, m.cmdCompleteTask(id)
		}
		return m, nil

	case key.Matches(msg, Keys.Cancel):
		if m.selectedTask != nil {
			id := m.selectedTask.ID
			return m, m.cmdCancelTask(id)
		}
		return m, nil

	case key.Matches(msg, Keys.Delete):
		if m.selectedTask != nil {
			id := m.selectedTask.ID
			title := m.selectedTask.Title
			m.openConfirm("Delete \""+title+"\"?", func() tea.Cmd {
				return m.cmdDeleteTask(id)
			})
		}
		return m, nil

	case key.Matches(msg, Keys.Edit):
		if m.selectedTask != nil {
			m.editing = true
			m.editInput.SetValue(m.selectedTask.Title)
			m.editInput.Focus()
		}
		return m, nil

	case key.Matches(msg, Keys.Schedule):
		if m.selectedTask != nil {
			m.openSchedule(m.selectedTask.ID)
		}
		return m, nil

	case key.Matches(msg, Keys.Move):
		if m.selectedTask != nil {
			taskID := m.selectedTask.ID
			pickerItems := []PickerItem{{ID: "", Label: "No project (Inbox)"}}
			for _, p := range m.projects {
				pickerItems = append(pickerItems, PickerItem{ID: p.ID, Label: p.Title})
			}
			m.openPicker("Move to Project", pickerItems, func(id string) tea.Cmd {
				if id == "" {
					return m.cmdMoveToProject(taskID, nil)
				}
				return m.cmdMoveToProject(taskID, &id)
			})
		}
		return m, nil
	}

	return m, nil
}

// ── Detail pane keys ───────────────────────────────────────────────────────────

func (m Model) updateDetail(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Tab switching and escape take priority over tab-specific keys.
	switch {
	case key.Matches(msg, Keys.Tab1):
		m.detailTab = TabNotes
		return m, nil
	case key.Matches(msg, Keys.Tab2):
		m.detailTab = TabChecklist
		return m, nil
	case key.Matches(msg, Keys.Tab3):
		m.detailTab = TabActivity
		return m, nil
	case key.Matches(msg, Keys.Escape):
		m.focusedPane = ListPane
		return m, nil
	}

	switch m.detailTab {
	case TabNotes:
		return m.updateDetailNotes(msg)
	case TabChecklist:
		return m.updateDetailChecklist(msg)
	case TabActivity:
		return m.updateDetailActivity(msg)
	}

	return m, nil
}

func (m Model) updateDetailNotes(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case key.Matches(msg, Keys.Down):
		m.detailScroll++
	case key.Matches(msg, Keys.Edit):
		m.statusFlash = "notes editing not implemented"
	}
	return m, nil
}

func (m Model) updateDetailChecklist(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.checkCursor > 0 {
			m.checkCursor--
		}
		return m, nil

	case key.Matches(msg, Keys.Down):
		if m.checkCursor < len(m.checklist)-1 {
			m.checkCursor++
		}
		return m, nil

	case key.Matches(msg, Keys.Complete):
		if m.selectedTask != nil && m.checkCursor < len(m.checklist) {
			item := m.checklist[m.checkCursor]
			taskID := m.selectedTask.ID
			// Status 0 = open, non-zero = completed.
			if item.Status != 0 {
				return m, m.cmdUncompleteCheckItem(taskID, item.ID)
			}
			return m, m.cmdCompleteCheckItem(taskID, item.ID)
		}
		return m, nil

	case key.Matches(msg, Keys.New):
		if m.selectedTask != nil {
			taskID := m.selectedTask.ID
			m.openInput("Checklist item:", func(title string) tea.Cmd {
				if title == "" {
					return nil
				}
				return m.cmdAddCheckItem(taskID, title)
			})
		}
		return m, nil

	case key.Matches(msg, Keys.Delete):
		if m.selectedTask != nil && m.checkCursor < len(m.checklist) {
			taskID := m.selectedTask.ID
			itemID := m.checklist[m.checkCursor].ID
			return m, m.cmdDeleteCheckItem(taskID, itemID)
		}
		return m, nil
	}

	return m, nil
}

func (m Model) updateDetailActivity(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Up):
		if m.detailScroll > 0 {
			m.detailScroll--
		}
		return m, nil

	case key.Matches(msg, Keys.Down):
		m.detailScroll++
		return m, nil

	case key.Matches(msg, Keys.Comment):
		if m.selectedTask != nil {
			taskID := m.selectedTask.ID
			m.openInput("Add comment:", func(content string) tea.Cmd {
				if content == "" {
					return nil
				}
				return m.cmdAddComment(taskID, content)
			})
		}
		return m, nil
	}

	return m, nil
}

// ── Inline editing ─────────────────────────────────────────────────────────────

func (m Model) updateEditing(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, Keys.Enter):
		if m.selectedTask != nil {
			id := m.selectedTask.ID
			newTitle := m.editInput.Value()
			m.editing = false
			m.editInput.Blur()
			return m, m.cmdUpdateTitle(id, newTitle)
		}
		m.editing = false
		m.editInput.Blur()
		return m, nil

	case key.Matches(msg, Keys.Escape):
		m.editing = false
		m.editInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}
