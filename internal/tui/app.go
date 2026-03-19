package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atask/atask/internal/client"
)

const (
	PaneSidebar = iota
	PaneList
	PaneDetail
)

// Stub types — will be fully implemented in separate files.
type List struct{ height, width int }
type Palette struct{}
type Search struct{}
type Help struct{}
type Confirm struct{}

func (l List) View() string     { return "[list]" }
func (p Palette) View() string  { return "[palette]" }
func (s Search) View() string   { return "[search]" }
func (h Help) View() string     { return "[help]" }
func (c Confirm) View() string  { return "[confirm]" }

func (l List) Update(msg tea.Msg) tea.Cmd     { return nil }
func (p Palette) Update(msg tea.Msg) tea.Cmd  { return nil }
func (s Search) Update(msg tea.Msg) tea.Cmd   { return nil }
func (h Help) Update(msg tea.Msg) tea.Cmd     { return nil }
func (c Confirm) Update(msg tea.Msg) tea.Cmd  { return nil }

// App is the root bubbletea model coordinating three panes.
type App struct {
	client *client.Client
	width  int
	height int
	focus  int // PaneSidebar, PaneList, PaneDetail

	sidebar   Sidebar
	list      List
	detail    Detail
	statusbar StatusBar

	// Overlay state (nil = not shown)
	palette *Palette
	search  *Search
	help    *Help
	confirm *Confirm

	// Cached data
	areas     []client.Area
	projects  []client.Project
	tags      []client.Tag
	locations []client.Location

	// SSE channel
	sseEvents <-chan client.DomainEvent

	err error
}

// NewApp constructs an App with the given API client.
func NewApp(c *client.Client) App {
	return App{
		client:  c,
		focus:   PaneSidebar,
		sidebar: NewSidebar(),
		detail:  NewDetail(),
		statusbar: StatusBar{
			context: "Inbox",
		},
	}
}

// Init returns a batch of commands to load initial data and start SSE.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.cmdLoadAreas(),
		a.cmdLoadProjects(),
		a.cmdLoadTags(),
		a.cmdLoadLocations(),
		a.cmdLoadInbox(),
		a.cmdStartSSE(),
	)
}

// Update handles all incoming messages.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a = a.resizePanes()
		return a, nil

	case tea.KeyPressMsg:
		// Route to active overlay first.
		if a.palette != nil {
			cmd := a.palette.Update(msg)
			if isEscape(msg) {
				a.palette = nil
			}
			return a, cmd
		}
		if a.search != nil {
			cmd := a.search.Update(msg)
			if isEscape(msg) {
				a.search = nil
			}
			return a, cmd
		}
		if a.help != nil {
			cmd := a.help.Update(msg)
			if isEscape(msg) {
				a.help = nil
			}
			return a, cmd
		}
		if a.confirm != nil {
			cmd := a.confirm.Update(msg)
			if isEscape(msg) {
				a.confirm = nil
			}
			return a, cmd
		}

		// Global keys.
		switch {
		case isQuit(msg):
			return a, tea.Quit
		case isTab(msg):
			a.focus = (a.focus + 1) % 3
			return a, nil
		case isShiftTab(msg):
			a.focus = (a.focus + 2) % 3
			return a, nil
		case isRune(msg, ':'):
			p := &Palette{}
			a.palette = p
			return a, nil
		case isRune(msg, '/'):
			s := &Search{}
			a.search = s
			return a, nil
		case isRune(msg, '?'):
			h := &Help{}
			a.help = h
			return a, nil
		case isRune(msg, 'r'):
			return a, a.cmdLoadInbox()
		}

		// Delegate to focused pane.
		var cmd tea.Cmd
		switch a.focus {
		case PaneSidebar:
			var newSidebar Sidebar
			newSidebar, cmd = a.sidebar.Update(msg)
			a.sidebar = newSidebar
		case PaneList:
			cmd = a.list.Update(msg)
		case PaneDetail:
			var detailCmd tea.Cmd
			a.detail, detailCmd = a.detail.Update(msg)
			cmd = detailCmd
		}
		return a, cmd

	// --- Data loaded messages ---

	case AreasLoadedMsg:
		a.areas = msg.Areas
		return a, nil

	case ProjectsLoadedMsg:
		a.projects = msg.Projects
		return a, nil

	case TagsLoadedMsg:
		a.tags = msg.Tags
		return a, nil

	case LocationsLoadedMsg:
		a.locations = msg.Locations
		return a, nil

	case TasksLoadedMsg:
		// Propagate tasks to list pane (stub: no-op for now).
		return a, nil

	// --- SSE messages ---

	case SSEEventMsg:
		// Trigger a targeted refresh based on event type, then keep listening.
		var refreshCmd tea.Cmd
		switch {
		case strings.HasPrefix(msg.Event.Type, "task."):
			refreshCmd = a.cmdLoadInbox()
		case strings.HasPrefix(msg.Event.Type, "project."):
			refreshCmd = a.cmdLoadProjects()
		case strings.HasPrefix(msg.Event.Type, "area."):
			refreshCmd = a.cmdLoadAreas()
		case strings.HasPrefix(msg.Event.Type, "tag."):
			refreshCmd = a.cmdLoadTags()
		}
		return a, tea.Batch(refreshCmd, a.cmdListenSSE())

	case sseStartedMsg:
		a.sseEvents = msg.ch
		return a, a.cmdListenSSE()

	case SSEDisconnectedMsg:
		a.statusbar.err = "SSE disconnected — reconnecting…"
		return a, a.cmdStartSSE()

	// --- UI control messages ---

	case ErrorMsg:
		a.statusbar.err = msg.Err.Error()
		a.err = msg.Err
		return a, nil

	case FlashMsg:
		a.statusbar.flash = msg.Message
		return a, nil

	case ClearFlashMsg:
		a.statusbar.flash = ""
		a.statusbar.err = ""
		return a, nil

	case FocusPaneMsg:
		if msg.Pane >= PaneSidebar && msg.Pane <= PaneDetail {
			a.focus = msg.Pane
		}
		return a, nil
	}

	return a, nil
}

// View renders the three-pane layout with the status bar below.
func (a App) View() tea.View {
	sidebarView := a.paneView(PaneSidebar, a.sidebar.View())
	listView := a.paneView(PaneList, a.list.View())
	detailView := a.paneView(PaneDetail, a.detail.View())

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, listView, detailView)
	full := lipgloss.JoinVertical(lipgloss.Left, main, a.statusbar.View())

	// Render overlays centered on top.
	if a.palette != nil {
		return tea.NewView(a.renderOverlay(full, a.palette.View()))
	}
	if a.search != nil {
		return tea.NewView(a.renderOverlay(full, a.search.View()))
	}
	if a.help != nil {
		return tea.NewView(a.renderOverlay(full, a.help.View()))
	}
	if a.confirm != nil {
		return tea.NewView(a.renderOverlay(full, a.confirm.View()))
	}

	return tea.NewView(full)
}

// paneView wraps a pane's content with a focused or blurred border.
func (a App) paneView(pane int, content string) string {
	if a.focus == pane {
		return FocusedBorder.Render(content)
	}
	return BlurredBorder.Render(content)
}

// renderOverlay places overlay content centered on top of the base view.
func (a App) renderOverlay(base, overlay string) string {
	rendered := OverlayStyle.Render(overlay)
	// Simple approach: place the overlay at the top of the base view.
	// A full centered overlay would require terminal cell manipulation.
	lines := strings.Split(base, "\n")
	overlayLines := strings.Split(rendered, "\n")

	// Center horizontally.
	overlayWidth := lipgloss.Width(rendered)
	leftPad := (a.width - overlayWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	padding := strings.Repeat(" ", leftPad)

	// Center vertically: start at 1/4 of the screen.
	startRow := (a.height - len(overlayLines)) / 4
	if startRow < 0 {
		startRow = 0
	}

	for i, ol := range overlayLines {
		row := startRow + i
		if row < len(lines) {
			// Replace the center section of the base line with the overlay line.
			base_line := lines[row]
			base_runes := []rune(base_line)
			overlay_runes := []rune(padding + ol)
			if leftPad+len([]rune(ol)) <= len(base_runes) {
				merged := string(base_runes[:leftPad]) + string(overlay_runes[leftPad:]) + string(base_runes[leftPad+len([]rune(ol)):])
				lines[row] = merged
			} else {
				lines[row] = padding + ol
			}
		}
	}
	return strings.Join(lines, "\n")
}

// resizePanes distributes available space among the three panes.
func (a App) resizePanes() App {
	// Status bar height is 1 line.
	const statusBarHeight = 1
	// Each border adds 2 (top + bottom) rows and 2 (left + right) columns.
	const borderSize = 2

	contentHeight := a.height - statusBarHeight - borderSize
	if contentHeight < 0 {
		contentHeight = 0
	}

	// Sidebar has a fixed width; list and detail share the rest equally.
	sidebarContentWidth := SidebarWidth
	remaining := a.width - (sidebarContentWidth + borderSize) - borderSize*2
	if remaining < 0 {
		remaining = 0
	}
	listWidth := remaining / 2
	detailWidth := remaining - listWidth

	a.sidebar.SetSize(sidebarContentWidth, contentHeight)
	a.list.width = listWidth
	a.list.height = contentHeight
	a.detail.SetSize(detailWidth, contentHeight)
	a.statusbar.width = a.width

	return a
}

// --- Commands ---

func (a App) cmdLoadAreas() tea.Cmd {
	return func() tea.Msg {
		areas, err := a.client.ListAreas(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return AreasLoadedMsg{Areas: areas}
	}
}

func (a App) cmdLoadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := a.client.ListProjects(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return ProjectsLoadedMsg{Projects: projects}
	}
}

func (a App) cmdLoadTags() tea.Cmd {
	return func() tea.Msg {
		tags, err := a.client.ListTags(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TagsLoadedMsg{Tags: tags}
	}
}

func (a App) cmdLoadLocations() tea.Cmd {
	return func() tea.Msg {
		locations, err := a.client.ListLocations(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return LocationsLoadedMsg{Locations: locations}
	}
}

func (a App) cmdLoadInbox() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListInbox(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdStartSSE() tea.Cmd {
	return func() tea.Msg {
		ch, err := a.client.SubscribeEvents(context.Background(), "tasks,projects,areas,tags")
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return sseStartedMsg{ch: ch}
	}
}

// sseStartedMsg is an internal message carrying the SSE channel.
type sseStartedMsg struct {
	ch <-chan client.DomainEvent
}

// cmdListenSSE returns a command that reads the next event from the SSE channel.
func (a App) cmdListenSSE() tea.Cmd {
	if a.sseEvents == nil {
		return nil
	}
	ch := a.sseEvents
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return SSEDisconnectedMsg{}
		}
		return SSEEventMsg{Event: evt}
	}
}
