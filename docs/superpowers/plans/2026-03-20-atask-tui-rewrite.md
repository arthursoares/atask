# atask TUI Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite `internal/tui/` with single-model architecture, proper layout math, `key.Binding`, centralized scroll, and rate-limited rendering — same features, better internals.

**Architecture:** Single `Model` struct with `focusedPane` enum. Render functions receive explicit `(width, height)`. Overlays are state fields (nil = hidden). All API calls are `tea.Cmd` functions in `commands.go`. `applyScroll` with indicators. `key.Binding` from bubbles for help integration.

**Tech Stack:** Bubbletea v2 (`charm.land/bubbletea/v2`), Bubbles v2 (`charm.land/bubbles/v2`), Lipgloss v2 (`charm.land/lipgloss/v2`)

**Spec:** `docs/superpowers/specs/2026-03-19-atask-tui-rewrite.md`

**Reference:** metering-tui at `../sw__antigravity/go/cmd/metering-tui/` — read `app.go`, `styles.go`, `keys.go`, `status_bar.go`, `scroll.go` for patterns.

---

## File Map

**Delete all existing files except `login.go`:**
```bash
# Files to delete (replaced by new architecture):
internal/tui/app.go
internal/tui/confirm.go
internal/tui/detail.go
internal/tui/help.go
internal/tui/keys.go
internal/tui/list.go
internal/tui/messages.go
internal/tui/palette.go
internal/tui/picker.go
internal/tui/schedule.go
internal/tui/search.go
internal/tui/sidebar.go
internal/tui/statusbar.go
internal/tui/styles.go
```

**Create new files:**
```
internal/tui/
├── model.go      → Model struct, Pane/DetailTab enums, NewModel, Init
├── update.go     → Update, updateSidebar, updateList, updateDetail, updateEditing
├── view.go       → View, renderHeader, renderSidebar, renderList, renderDetail, renderFooter
├── overlay.go    → overlay state types, update funcs, render funcs (palette, search, picker, confirm, schedule, input)
├── commands.go   → all tea.Cmd funcs (load data, SSE, API mutations, refresh)
├── keys.go       → key.Binding definitions using bubbles/key
├── styles.go     → color palette consts, pre-rendered chars, lipgloss styles
├── messages.go   → all tea.Msg types
├── scroll.go     → applyScroll, renderPane helpers
├── login.go      → KEEP UNCHANGED
```

---

## Phase 1: Foundation (styles, keys, messages, scroll)

### Task 1: Delete old TUI files and create styles.go + scroll.go

**Files:**
- Delete: all `internal/tui/*.go` except `login.go`
- Create: `internal/tui/styles.go`
- Create: `internal/tui/scroll.go`

- [ ] **Step 1: Delete old files**

```bash
cd internal/tui
ls *.go | grep -v login.go | xargs rm
cd ../..
```

- [ ] **Step 2: Create styles.go**

```go
package tui

import "charm.land/lipgloss/v2"

// Color palette.
const (
	ColorPrimary   = "#7C3AED" // Purple — focused borders, agent actor
	ColorSecondary = "#38BDF8" // Cyan — selected items
	ColorSuccess   = "#22C55E" // Green — completed
	ColorWarning   = "#F59E0B" // Orange — upcoming deadlines
	ColorError     = "#EF4444" // Red — overdue, errors
	ColorMuted     = "#6B7280" // Gray — dimmed text, unfocused borders
	ColorBg        = "#1E293B" // Dark blue — selected row background
)

// Layout constants.
const (
	SidebarWidth = 22
	HeaderHeight = 1
	FooterHeight = 1
	BorderCols   = 2 // per pane (left + right)
	BorderRows   = 2 // per pane (top + bottom)
	NumPanes     = 3
)

// Pane borders.
var (
	FocusedBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPrimary))
	BlurredBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorMuted))
)

// Text styles.
var (
	TitleStyle    = lipgloss.NewStyle().Bold(true)
	MutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	ErrorTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	SelectedStyle = lipgloss.NewStyle().Background(lipgloss.Color(ColorBg)).Bold(true)
	ActiveTabStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSecondary)).Underline(true)
	InactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	AgentStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPrimary))
	HumanStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary))
	OverdueStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError))
	HeaderBarStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorSecondary))
	FooterBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorMuted))
	OverlayBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorSecondary)).
			Padding(1, 2)
)

// Pre-rendered characters (avoid allocation in render loops).
var (
	CheckOpen   = MutedStyle.Render("☐")
	CheckDone   = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSuccess)).Render("✓")
	CheckCancel = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorError)).Render("✗")
)

// View icons.
var ViewIcons = map[string]string{
	"inbox":    "📥",
	"today":    "⭐",
	"upcoming": "📅",
	"someday":  "💤",
	"logbook":  "📓",
}
```

- [ ] **Step 3: Create scroll.go**

```go
package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// applyScroll clips lines to the visible window and returns a scroll indicator.
func applyScroll(lines []string, offset, visibleHeight int) (visible []string, indicator string) {
	total := len(lines)
	if total <= visibleHeight {
		// Pad to full height.
		for len(lines) < visibleHeight {
			lines = append(lines, "")
		}
		return lines, ""
	}
	if offset < 0 {
		offset = 0
	}
	if offset > total-visibleHeight {
		offset = total - visibleHeight
	}
	visible = lines[offset : offset+visibleHeight]

	indicator = fmt.Sprintf("%d–%d of %d", offset+1, offset+visibleHeight, total)
	if offset > 0 {
		indicator = "↑ " + indicator
	}
	if offset+visibleHeight < total {
		indicator += " ↓"
	}
	return visible, MutedStyle.Render(indicator)
}

// renderPane wraps content in a border with the given dimensions.
func renderPane(content string, width, height int, focused bool) string {
	border := BlurredBorder
	if focused {
		border = FocusedBorder
	}
	return border.Width(width).Height(height).Render(content)
}

// truncateWithEllipsis truncates s to maxWidth, adding "…" if truncated.
func truncateWithEllipsis(s string, maxWidth int) string {
	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}
	// Crude truncation — works for ASCII. For full Unicode, use runewidth.
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxWidth-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}

// padRight pads s with spaces to exactly width.
func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./internal/tui/`
Expected: may fail because login.go references types from deleted files. That's expected — we'll fix in Task 2.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: delete old TUI files, add styles.go and scroll.go"
```

### Task 2: Create keys.go and messages.go

**Files:**
- Create: `internal/tui/keys.go`
- Create: `internal/tui/messages.go`

- [ ] **Step 1: Create keys.go**

```go
package tui

import "charm.land/bubbles/v2/key"

// Keys defines all key bindings for the TUI.
var Keys = struct {
	Up, Down      key.Binding
	Enter, Escape key.Binding
	Tab, ShiftTab key.Binding
	Top, Bottom   key.Binding

	New, Edit, Complete key.Binding
	Cancel, Delete      key.Binding
	Schedule, Move, Tag key.Binding
	Location, Comment   key.Binding

	Palette, Search, Help key.Binding
	Refresh, Quit         key.Binding

	Tab1, Tab2, Tab3 key.Binding
}{
	Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
	Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
	Enter:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Escape:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
	Top:      key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:   key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),

	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Complete: key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "complete")),
	Cancel:   key.NewBinding(key.WithKeys("X"), key.WithHelp("X", "cancel")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Schedule: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "schedule")),
	Move:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move")),
	Tag:      key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tag")),
	Location: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "location")),
	Comment:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "comment")),

	Palette: key.NewBinding(key.WithKeys(":", "ctrl+p"), key.WithHelp(":/ctrl+p", "commands")),
	Search:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
	Quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),

	Tab1: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "notes")),
	Tab2: key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "checklist")),
	Tab3: key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "activity")),
}
```

- [ ] **Step 2: Create messages.go**

```go
package tui

import "github.com/atask/atask/internal/client"

// Data loaded from API.
type (
	TasksLoadedMsg     struct{ Tasks []client.Task }
	AreasLoadedMsg     struct{ Areas []client.Area }
	ProjectsLoadedMsg  struct{ Projects []client.Project }
	TagsLoadedMsg      struct{ Tags []client.Tag }
	LocationsLoadedMsg struct{ Locations []client.Location }
	SectionsLoadedMsg  struct{ Sections []client.Section }
	ChecklistLoadedMsg struct{ Items []client.ChecklistItem }
	ActivitiesLoadedMsg struct{ Activities []client.Activity }
)

// Mutation results.
type (
	TaskCreatedMsg   struct{ Task client.Task }
	TaskCompletedMsg struct{ ID string }
	TaskDeletedMsg   struct{ ID string }
	RefreshMsg       struct{}
	ErrorMsg         struct{ Err error }
	FlashMsg         struct{ Message string }
	ClearFlashMsg    struct{}
)

// SSE.
type (
	SSEStartedMsg      struct{ Events <-chan client.DomainEvent }
	SSEEventMsg        struct{ Event client.DomainEvent }
	SSEDisconnectedMsg struct{}
)

// Detail refresh (internal).
type DetailRefreshMsg struct{ TaskID string }
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/tui/`
Expected: still may fail (login.go deps). That's fine.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/keys.go internal/tui/messages.go
git commit -m "feat: add key bindings and message types for TUI rewrite"
```

---

## Phase 2: Model, Commands, and Core Update

### Task 3: Create model.go with Model struct and Init

**Files:**
- Create: `internal/tui/model.go`

- [ ] **Step 1: Implement model.go**

Define all enums, overlay state types, the `Model` struct, `NewModel`, and `Init`.

```go
package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/atask/atask/internal/client"
)

// Pane identifies which pane has focus.
type Pane int

const (
	SidebarPane Pane = iota
	ListPane
	DetailPane
)

// DetailTab identifies which tab is active in the detail pane.
type DetailTab int

const (
	TabNotes DetailTab = iota
	TabChecklist
	TabActivity
)

// SidebarItem represents one row in the sidebar.
type SidebarItem struct {
	Label    string
	Kind     string // "view", "area", "project", "tag", "header"
	ID       string
	Count    int
	Indent   int
	Expanded bool
}

// Overlay state types.
type PaletteState struct {
	input    textinput.Model
	commands []PaletteCommand
	filtered []PaletteCommand
	cursor   int
}

type PaletteCommand struct {
	Name     string
	Category string
	Action   func() tea.Cmd
}

type SearchState struct {
	input textinput.Model
}

type ConfirmState struct {
	message string
	onYes   func() tea.Cmd
}

type PickerState struct {
	title    string
	input    textinput.Model
	items    []PickerItem
	filtered []PickerItem
	cursor   int
	onSelect func(id string) tea.Cmd
}

type PickerItem struct {
	ID    string
	Label string
}

type ScheduleState struct {
	taskID  string
	options []string
	cursor  int
}

type InputState struct {
	prompt string
	input  textinput.Model
	onDone func(value string) tea.Cmd
}

// Model is the single root model for the TUI.
type Model struct {
	client      *client.Client
	width       int
	height      int
	focusedPane Pane

	// Sidebar.
	sidebarItems  []SidebarItem
	sidebarCursor int
	sidebarScroll int
	areaExpanded  map[string]bool

	// List.
	tasks      []client.Task
	listCursor int
	listScroll int
	listTitle  string

	// Detail.
	selectedTask *client.Task
	detailTab    DetailTab
	checklist    []client.ChecklistItem
	checkCursor  int
	checkScroll  int
	activities   []client.Activity
	detailScroll int

	// Cached data.
	areas       []client.Area
	projects    []client.Project
	tags        []client.Tag
	locations   []client.Location
	currentView string

	// Overlays (nil = hidden).
	palette     *PaletteState
	search      *SearchState
	help        bool
	confirm     *ConfirmState
	picker      *PickerState
	schedule    *ScheduleState
	inputPrompt *InputState

	// Inline editing.
	editing   bool
	editInput textinput.Model

	// SSE.
	sseEvents <-chan client.DomainEvent

	// Status bar.
	statusContext string
	statusFlash   string
	statusErr     string

	// Render cache.
	lastRender  int64 // unix nano
	renderCache string
}

// NewModel creates the TUI model connected to the given API client.
func NewModel(c *client.Client) Model {
	return Model{
		client:       c,
		focusedPane:  SidebarPane,
		currentView:  "inbox",
		statusContext: "Inbox",
		areaExpanded: make(map[string]bool),
	}
}

// Init returns commands to load initial data and start SSE.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.cmdLoadAreas(),
		m.cmdLoadProjects(),
		m.cmdLoadTags(),
		m.cmdLoadLocations(),
		m.cmdLoadInbox(),
		m.cmdStartSSE(),
	)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: fails — `cmdLoad*` not defined yet (commands.go). That's expected.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/model.go
git commit -m "feat: add Model struct with single-model architecture"
```

### Task 4: Create commands.go

**Files:**
- Create: `internal/tui/commands.go`

- [ ] **Step 1: Implement commands.go**

All `tea.Cmd` functions that call the HTTP client. Every API interaction lives here.

```go
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/atask/atask/internal/client"
)

func (m Model) cmdLoadInbox() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListInbox(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (m Model) cmdLoadToday() tea.Cmd { /* same pattern with ListToday */ }
func (m Model) cmdLoadUpcoming() tea.Cmd { /* ListUpcoming */ }
func (m Model) cmdLoadSomeday() tea.Cmd { /* ListSomeday */ }
func (m Model) cmdLoadLogbook() tea.Cmd { /* ListLogbook */ }

func (m Model) cmdLoadProjectTasks(projectID string) tea.Cmd { /* ListTasksByProject */ }
func (m Model) cmdLoadAreaTasks(areaID string) tea.Cmd { /* ListTasksByArea */ }

func (m Model) cmdLoadAreas() tea.Cmd { /* ListAreas → AreasLoadedMsg */ }
func (m Model) cmdLoadProjects() tea.Cmd { /* ListProjects → ProjectsLoadedMsg */ }
func (m Model) cmdLoadTags() tea.Cmd { /* ListTags → TagsLoadedMsg */ }
func (m Model) cmdLoadLocations() tea.Cmd { /* ListLocations → LocationsLoadedMsg */ }

func (m Model) cmdLoadChecklist(taskID string) tea.Cmd { /* ListChecklistItems → ChecklistLoadedMsg */ }
func (m Model) cmdLoadActivities(taskID string) tea.Cmd { /* ListActivities → ActivitiesLoadedMsg */ }

func (m Model) cmdCompleteTask(id string) tea.Cmd { /* CompleteTask → RefreshMsg */ }
func (m Model) cmdCancelTask(id string) tea.Cmd { /* CancelTask → RefreshMsg */ }
func (m Model) cmdDeleteTask(id string) tea.Cmd { /* DeleteTask → RefreshMsg */ }
func (m Model) cmdCreateTask(title string) tea.Cmd { /* CreateTask → TaskCreatedMsg */ }
func (m Model) cmdUpdateTitle(id, title string) tea.Cmd { /* UpdateTaskTitle → RefreshMsg */ }
func (m Model) cmdUpdateSchedule(id, schedule string) tea.Cmd { /* UpdateTaskSchedule → RefreshMsg */ }
func (m Model) cmdMoveToProject(id string, projectID *string) tea.Cmd { /* MoveTaskToProject → RefreshMsg */ }
func (m Model) cmdAddComment(taskID, content string) tea.Cmd { /* AddActivity → DetailRefreshMsg */ }
func (m Model) cmdCompleteCheckItem(taskID, itemID string) tea.Cmd { /* CompleteChecklistItem → DetailRefreshMsg */ }
func (m Model) cmdUncompleteCheckItem(taskID, itemID string) tea.Cmd { /* UncompleteChecklistItem → DetailRefreshMsg */ }
func (m Model) cmdAddCheckItem(taskID, title string) tea.Cmd { /* AddChecklistItem → DetailRefreshMsg */ }
func (m Model) cmdDeleteCheckItem(taskID, itemID string) tea.Cmd { /* DeleteChecklistItem → DetailRefreshMsg */ }

// refreshCurrentView returns the right load command for the active view.
func (m Model) refreshCurrentView() tea.Cmd {
	switch {
	case m.currentView == "" || m.currentView == "inbox":
		return m.cmdLoadInbox()
	case m.currentView == "today":
		return m.cmdLoadToday()
	case m.currentView == "upcoming":
		return m.cmdLoadUpcoming()
	case m.currentView == "someday":
		return m.cmdLoadSomeday()
	case m.currentView == "logbook":
		return m.cmdLoadLogbook()
	case strings.HasPrefix(m.currentView, "project:"):
		return m.cmdLoadProjectTasks(strings.TrimPrefix(m.currentView, "project:"))
	case strings.HasPrefix(m.currentView, "area:"):
		return m.cmdLoadAreaTasks(strings.TrimPrefix(m.currentView, "area:"))
	default:
		return m.cmdLoadInbox()
	}
}

// refreshDetail reloads checklist and activities for the selected task.
func (m Model) refreshDetail() tea.Cmd {
	if m.selectedTask == nil {
		return nil
	}
	id := m.selectedTask.ID
	return tea.Batch(m.cmdLoadChecklist(id), m.cmdLoadActivities(id))
}

// SSE lifecycle.
func (m Model) cmdStartSSE() tea.Cmd {
	return func() tea.Msg {
		events, err := m.client.SubscribeEvents(context.Background(), "*")
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return SSEStartedMsg{Events: events}
	}
}

func cmdListenSSE(events <-chan client.DomainEvent) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-events
		if !ok {
			return SSEDisconnectedMsg{}
		}
		return SSEEventMsg{Event: evt}
	}
}

// handleSSEEvent decides what to refresh based on event type.
func (m Model) handleSSEEvent(evt client.DomainEvent) tea.Cmd {
	prefix := strings.Split(evt.Type, ".")[0]
	switch prefix {
	case "task":
		return m.refreshCurrentView()
	case "project":
		return tea.Batch(m.cmdLoadProjects(), m.refreshCurrentView())
	case "area":
		return tea.Batch(m.cmdLoadAreas(), m.refreshCurrentView())
	case "checklist", "activity":
		return m.refreshDetail()
	case "tag":
		return m.cmdLoadTags()
	case "location":
		return m.cmdLoadLocations()
	}
	return nil
}
```

The implementer should fill in each function following the pattern — call `m.client.X(context.Background(), ...)`, return the appropriate message on success, `ErrorMsg` on error. Reference `internal/client/client.go` for exact method signatures.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: fails — update.go and view.go not yet created.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/commands.go
git commit -m "feat: add all API command functions for TUI"
```

---

## Phase 3: Update Logic

### Task 5: Create update.go

**Files:**
- Create: `internal/tui/update.go`

This is the biggest file. It contains `Update` plus per-pane update handlers.

- [ ] **Step 1: Implement update.go**

```go
package tui

import (
	"strings"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		// 1. Overlay intercepts.
		if m.palette != nil { return m.updatePalette(msg) }
		if m.search != nil { return m.updateSearch(msg) }
		if m.help { m.help = false; return m, nil }
		if m.confirm != nil { return m.updateConfirm(msg) }
		if m.picker != nil { return m.updatePicker(msg) }
		if m.schedule != nil { return m.updateSchedule(msg) }
		if m.inputPrompt != nil { return m.updateInput(msg) }
		if m.editing { return m.updateEditing(msg) }

		// 2. Global keys.
		switch {
		case key.Matches(msg, Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, Keys.Tab):
			m.cycleFocus(1)
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
			m.help = true
			return m, nil
		case key.Matches(msg, Keys.Refresh):
			return m, m.refreshCurrentView()
		}

		// 3. Pane-specific.
		switch m.focusedPane {
		case SidebarPane:
			return m.updateSidebar(msg)
		case ListPane:
			return m.updateList(msg)
		case DetailPane:
			return m.updateDetail(msg)
		}

	// Data loaded.
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

	case TaskCreatedMsg:
		return m, m.refreshCurrentView()

	case RefreshMsg:
		return m, m.refreshCurrentView()

	case DetailRefreshMsg:
		return m, m.refreshDetail()

	case ErrorMsg:
		m.statusErr = msg.Err.Error()
		return m, nil

	case FlashMsg:
		m.statusFlash = msg.Message
		return m, nil

	// SSE lifecycle.
	case SSEStartedMsg:
		m.sseEvents = msg.Events
		return m, cmdListenSSE(m.sseEvents)

	case SSEEventMsg:
		cmd := m.handleSSEEvent(msg.Event)
		return m, tea.Batch(cmd, cmdListenSSE(m.sseEvents))

	case SSEDisconnectedMsg:
		m.statusErr = "SSE disconnected"
		return m, nil
	}

	return m, nil
}

func (m *Model) cycleFocus(dir int) {
	m.focusedPane = Pane((int(m.focusedPane) + dir + NumPanes) % NumPanes)
}

func (m *Model) selectFirstTask() {
	if len(m.tasks) > 0 {
		m.selectedTask = &m.tasks[0]
	} else {
		m.selectedTask = nil
	}
}
```

**updateSidebar:** handle Up/Down (cursor movement), Enter (select view/toggle area/select project/select tag), New (create area/project), Edit (rename), Delete, Archive.

**updateList:** handle Up/Down, Enter (focus detail), New (open input prompt for task title), Complete, Cancel, Delete (open confirm), Edit (enter inline edit mode), Schedule (open schedule overlay), Move (open picker with projects).

**updateDetail:** handle Tab1/2/3 (switch tabs), Up/Down (scroll content or navigate checklist), Complete (toggle checklist item), New (add checklist item), Delete (remove checklist item), Comment (add activity), Edit (notes — flash "not implemented" for v0).

**updateEditing:** handle Enter (save title), Escape (cancel), all other keys → textinput.

**rebuildSidebar:** rebuild `m.sidebarItems` from `m.areas`, `m.projects`, `m.tags`, respecting `m.areaExpanded`.

Each of these is a method on `Model` returning `(tea.Model, tea.Cmd)`.

The implementer should reference the spec for exact key mappings per pane and the current `update.go`/`app.go` for the interaction logic (what message to emit, what overlay to open, etc.).

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`

- [ ] **Step 3: Commit**

```bash
git add internal/tui/update.go
git commit -m "feat: add Update logic with pane-specific handlers"
```

---

## Phase 4: View Rendering

### Task 6: Create view.go

**Files:**
- Create: `internal/tui/view.go`

- [ ] **Step 1: Implement view.go**

```go
package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) View() tea.View {
	// Rate limit.
	now := time.Now().UnixNano()
	if now-m.lastRender < 50_000_000 && m.renderCache != "" {
		return tea.NewView(m.renderCache)
	}

	// Dimensions.
	contentHeight := m.height - HeaderHeight - FooterHeight - BorderRows
	if contentHeight < 1 { contentHeight = 1 }
	remaining := m.width - SidebarWidth - (NumPanes * BorderCols)
	if remaining < 2 { remaining = 2 }
	listWidth := remaining * 3 / 10
	if listWidth < 10 { listWidth = 10 }
	detailWidth := remaining - listWidth

	// Render panes.
	sidebar := renderPane(m.renderSidebar(SidebarWidth, contentHeight), SidebarWidth, contentHeight, m.focusedPane == SidebarPane)
	list := renderPane(m.renderList(listWidth, contentHeight), listWidth, contentHeight, m.focusedPane == ListPane)
	detail := renderPane(m.renderDetail(detailWidth, contentHeight), detailWidth, contentHeight, m.focusedPane == DetailPane)

	// Compose.
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, list, detail)
	full := lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), content, m.renderFooter())

	// Overlay.
	if m.palette != nil { full = m.renderOverlay(full, m.renderPaletteOverlay()) }
	if m.search != nil { full = m.renderOverlay(full, m.renderSearchOverlay()) }
	if m.help { full = m.renderOverlay(full, m.renderHelpOverlay()) }
	if m.confirm != nil { full = m.renderOverlay(full, m.renderConfirmOverlay()) }
	if m.picker != nil { full = m.renderOverlay(full, m.renderPickerOverlay()) }
	if m.schedule != nil { full = m.renderOverlay(full, m.renderScheduleOverlay()) }
	if m.inputPrompt != nil { full = m.renderOverlay(full, m.renderInputOverlay()) }

	m.renderCache = full
	m.lastRender = now
	return tea.NewView(full)
}

func (m Model) renderHeader() string {
	left := HeaderBarStyle.Render(m.statusContext)
	var middle string
	if m.statusErr != "" {
		middle = ErrorTextStyle.Render(m.statusErr)
	} else if m.statusFlash != "" {
		middle = m.statusFlash
	}
	return lipgloss.NewStyle().Width(m.width).Render(
		left + "  " + middle,
	)
}

func (m Model) renderFooter() string {
	hints := FooterBarStyle.Render("[Tab] focus  [/] search  [:] cmd  [?] help  [q] quit")
	return lipgloss.NewStyle().Width(m.width).Render(hints)
}
```

**renderSidebar(width, height):** Build lines from `m.sidebarItems`. Each item rendered with icon, label (truncated to width), count badge right-aligned. Selected item highlighted. Apply scroll with `applyScroll`.

**renderList(width, height):** Title bar ("Today — 5 tasks") + task rows. Each row: checkbox + title (truncated) + right-aligned metadata (deadline, project icon). Selected row highlighted. Completed tasks dimmed. If `m.editing`, show textinput on selected row. Apply scroll.

**renderDetail(width, height):** If `m.selectedTask == nil`, show "Select a task". Otherwise: title (bold) + metadata line + tab bar + tab content + latest activity footer. Tab content scrollable with `applyScroll`.

**renderOverlay(base, overlay):** Center the overlay on the base. Split both into lines, replace base lines at center with overlay lines.

The implementer should reference the metering-tui's `renderBrowserPane`/`renderDataPane` in `status_bar.go` for the pattern of building lines with explicit width constraints using `lipgloss.NewStyle().Width(N).MaxWidth(N).Inline(true)`.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`

- [ ] **Step 3: Commit**

```bash
git add internal/tui/view.go
git commit -m "feat: add View rendering with explicit layout pipeline"
```

---

## Phase 5: Overlays

### Task 7: Create overlay.go

**Files:**
- Create: `internal/tui/overlay.go`

- [ ] **Step 1: Implement overlay.go**

All overlay update and render functions in one file. Each overlay follows the same pattern:
- `updateX(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)` — handle keys, close with Esc
- `renderXOverlay() string` — render the overlay content (without the border — `renderOverlay` in view.go wraps with `OverlayBorder`)

**Palette:** text input + filtered command list. Type to filter (subsequence match), j/k navigate, Enter executes, Esc closes.

**Search:** text input at top of list. Each keystroke updates `m.tasks` filter. Esc clears.

**Help:** static key reference text, scrollable. Esc closes.

**Confirm:** "Are you sure?" + y/n. y calls onYes callback, n/Esc closes.

**Picker:** text input + filterable item list. For projects/tags/locations. Enter selects, Esc closes.

**Schedule:** 3 options (Inbox/Today/Someday), j/k navigate, Enter selects, Esc closes.

**Input:** prompt + text input. For "New task title", "New area name", etc. Enter submits, Esc cancels.

Helper methods on Model:
- `openPalette()` — build commands list from context, create PaletteState
- `openSearch()` — create SearchState with focused textinput
- `openPicker(title, items, onSelect)` — create PickerState
- `openInput(prompt, onDone)` — create InputState
- `openSchedule(taskID)` — create ScheduleState
- `openConfirm(message, onYes)` — create ConfirmState

- [ ] **Step 2: Verify build**

Run: `go build ./internal/tui/`
Expected: PASS — all files now exist

- [ ] **Step 3: Run full test suite**

Run: `go test ./...`
Expected: all existing tests pass (TUI has no tests — just verify no compilation errors)

- [ ] **Step 4: Commit**

```bash
git add internal/tui/overlay.go
git commit -m "feat: add all overlay interactions (palette, search, picker, confirm, schedule, input)"
```

---

## Phase 6: Integration and Polish

### Task 8: Verify login.go compatibility and fix main.go

**Files:**
- Modify: `internal/tui/login.go` (if needed — check for broken references)
- Modify: `cmd/atask/main.go` (if needed — update `tui.NewApp` → `tui.NewModel`)

- [ ] **Step 1: Check login.go compiles**

Run: `go build ./internal/tui/`
If login.go references deleted types, fix them (it should be self-contained).

- [ ] **Step 2: Check main.go references**

The current main.go calls `tui.NewApp(c)`. Update to `tui.NewModel(c)`.

- [ ] **Step 3: Full build**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add cmd/atask/main.go internal/tui/login.go
git commit -m "fix: update main.go and login.go for new Model type"
```

### Task 9: Smoke test and polish

- [ ] **Step 1: Run all tests**

Run: `go test -race ./...`

- [ ] **Step 2: Build and smoke test**

```bash
go build -o /tmp/atask ./cmd/atask
/tmp/atask serve &
sleep 1
# Register + login
curl -s -X POST http://localhost:8080/auth/register -H 'Content-Type: application/json' -d '{"email":"demo@atask.dev","password":"secret","name":"Demo"}'
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login -H 'Content-Type: application/json' -d '{"email":"demo@atask.dev","password":"secret"}' | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
# Create some tasks
curl -s -X POST http://localhost:8080/tasks -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"title":"Test task 1"}'
curl -s -X POST http://localhost:8080/tasks -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"title":"Test task 2"}'
# Start TUI
ATASK_TOKEN=$TOKEN /tmp/atask
```

Verify:
- Three panes render without overflow
- Sidebar shows views, areas, tags
- Tasks appear in inbox
- Tab cycles panes
- j/k navigates
- : opens command palette
- / opens search
- ? opens help
- Selecting a task shows detail

- [ ] **Step 3: Fix any rendering issues found during smoke test**

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "fix: polish TUI rendering and layout"
```

### Task 10: Push and update docs

- [ ] **Step 1: Push branch**

Run: `git push --force-with-lease origin feat/tui`

- [ ] **Step 2: Done**
