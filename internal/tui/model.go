package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"github.com/atask/atask/internal/client"
)

// Pane identifies which pane has focus.
type Pane int

const (
	SidebarPane Pane = iota
	ListPane
	DetailPane
)

// DetailTab identifies the active tab in the detail pane.
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

// PaletteCommand is a single entry in the command palette.
type PaletteCommand struct {
	Name     string
	Category string
	Action   func() tea.Cmd
}

// PaletteState holds the state for the command palette overlay.
type PaletteState struct {
	Input    textinput.Model
	Commands []PaletteCommand
	Filtered []PaletteCommand
	Cursor   int
}

// SearchState holds the state for the search overlay.
type SearchState struct {
	Input textinput.Model
}

// ConfirmState holds the state for the confirmation dialog overlay.
type ConfirmState struct {
	Message string
	OnYes   func() tea.Cmd
}

// PickerItem is a single selectable entry in a picker overlay.
type PickerItem struct {
	ID    string
	Label string
}

// PickerState holds the state for a generic picker overlay.
type PickerState struct {
	Title    string
	Input    textinput.Model
	Items    []PickerItem
	Filtered []PickerItem
	Cursor   int
	OnSelect func(PickerItem) tea.Cmd
}

// ScheduleState holds the state for the schedule picker overlay.
type ScheduleState struct {
	TaskID  string
	Options []string
	Cursor  int
}

// InputState holds the state for a generic single-line input prompt overlay.
type InputState struct {
	Prompt  string
	Input   textinput.Model
	OnDone  func(string) tea.Cmd
}

// Model is the single top-level bubbletea model for the atask TUI.
type Model struct {
	// Infrastructure
	client *client.Client
	width  int
	height int

	// Pane focus
	focusedPane Pane

	// Sidebar state
	sidebarItems  []SidebarItem
	sidebarCursor int
	sidebarScroll int
	areaExpanded  map[string]bool

	// List state
	tasks      []client.Task
	listCursor int
	listScroll int
	listTitle  string

	// Detail state
	selectedTask *client.Task
	detailTab    DetailTab
	checklist    []client.ChecklistItem
	checkCursor  int
	activities   []client.Activity
	detailScroll int

	// Cached reference data
	areas       []client.Area
	projects    []client.Project
	tags        []client.Tag
	locations   []client.Location
	currentView string // "inbox", "today", "upcoming", "someday", "logbook", project ID, area ID, tag ID

	// Overlays — at most one is active at a time
	palette      *PaletteState
	search       *SearchState
	showHelp     bool
	confirm      *ConfirmState
	picker       *PickerState
	schedule     *ScheduleState
	inputPrompt  *InputState

	// Inline editing (task title)
	editing   bool
	editInput textinput.Model

	// SSE event channel (set after SSEStartedMsg)
	sseEvents <-chan client.DomainEvent

	// Status bar
	statusContext string // persistent context label (e.g. current view name)
	statusFlash   string // ephemeral flash message
	statusErr     string // error message

	// Render cache — skip re-render when model is unchanged
	lastRender  int64
	renderCache string
}

// NewModel returns a Model with sensible defaults ready to call Init.
func NewModel(c *client.Client) Model {
	editInput := textinput.New()

	return Model{
		client:       c,
		focusedPane:  SidebarPane,
		currentView:  "inbox",
		areaExpanded: make(map[string]bool),
		editInput:    editInput,
	}
}

// Init issues the initial batch of load commands.
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
