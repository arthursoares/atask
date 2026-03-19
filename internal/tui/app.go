package tui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/atask/atask/internal/client"
)

// inputOverlay is a lightweight single-line text-input overlay used for
// prompts like "New task title".
type inputOverlay struct {
	prompt string
	input  textinput.Model
	onDone func(value string) tea.Cmd
}

func newInputOverlay(prompt string, onDone func(string) tea.Cmd) inputOverlay {
	ti := textinput.New()
	ti.Placeholder = "…"
	ti.Focus()
	return inputOverlay{prompt: prompt, input: ti, onDone: onDone}
}

// Update handles key presses. Returns updated overlay, a command, and whether
// it was closed.
func (o inputOverlay) Update(msg tea.KeyPressMsg) (inputOverlay, tea.Cmd, bool) {
	switch {
	case isEscape(msg):
		return o, nil, true
	case isEnter(msg):
		val := strings.TrimSpace(o.input.Value())
		var cmd tea.Cmd
		if val != "" && o.onDone != nil {
			cmd = o.onDone(val)
		}
		return o, cmd, true
	default:
		var cmd tea.Cmd
		o.input, cmd = o.input.Update(msg)
		return o, cmd, false
	}
}

// View renders the input overlay content.
func (o inputOverlay) View() string {
	promptStyle := lipgloss.NewStyle().Bold(true)
	return promptStyle.Render(o.prompt) + "\n\n" + o.input.View()
}

const (
	PaneSidebar = iota
	PaneList
	PaneDetail
)

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
	palette      *Palette
	search       *Search
	help         *Help
	confirm      *Confirm
	schedule     *Schedule
	picker       *Picker
	inputOverlay *inputOverlay

	// Cached data
	areas     []client.Area
	projects  []client.Project
	tags      []client.Tag
	locations []client.Location

	// Active view: one of "inbox", "today", "upcoming", "someday", "logbook",
	// "project:<id>", or "area:<id>".
	currentView string

	// SSE channel
	sseEvents <-chan client.DomainEvent

	err error
}

// NewApp constructs an App with the given API client.
func NewApp(c *client.Client) App {
	return App{
		client:      c,
		focus:       PaneSidebar,
		sidebar:     NewSidebar(),
		list:        NewList(),
		detail:      NewDetail(),
		currentView: "inbox",
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
			updated, cmd, closed := a.palette.Update(msg)
			if closed {
				a.palette = nil
			} else {
				*a.palette = updated
			}
			return a, cmd
		}
		if a.search != nil {
			updated, cmd, closed := a.search.Update(msg)
			if closed {
				a.search = nil
			} else {
				*a.search = updated
			}
			return a, cmd
		}
		if a.help != nil {
			updated, cmd, closed := a.help.Update(msg)
			if closed {
				a.help = nil
			} else {
				*a.help = updated
			}
			return a, cmd
		}
		if a.confirm != nil {
			updated, cmd, closed := a.confirm.Update(msg)
			if closed {
				a.confirm = nil
			} else {
				*a.confirm = updated
			}
			return a, cmd
		}
		if a.schedule != nil {
			updated, cmd, closed := a.schedule.Update(msg)
			if closed {
				a.schedule = nil
			} else {
				*a.schedule = updated
			}
			return a, cmd
		}
		if a.picker != nil {
			updated, cmd, closed := a.picker.Update(msg)
			if closed {
				a.picker = nil
			} else {
				*a.picker = updated
			}
			return a, cmd
		}
		if a.inputOverlay != nil {
			updated, cmd, closed := a.inputOverlay.Update(msg)
			if closed {
				a.inputOverlay = nil
			} else {
				*a.inputOverlay = updated
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
		case isRune(msg, ':') || isCtrl(msg, 'p'):
			p := a.newPalette()
			a.palette = &p
			return a, nil
		case isRune(msg, '/'):
			s := NewSearch(a.list.width)
			a.search = &s
			return a, nil
		case isRune(msg, '?'):
			h := NewHelp(a.width, a.height)
			a.help = &h
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
			var newList List
			newList, cmd = a.list.Update(msg)
			a.list = newList
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
		a.list.SetTasks(msg.Tasks, a.statusbar.context)
		return a, nil

	case ChecklistLoadedMsg:
		a.detail.SetChecklist(msg.Items)
		return a, nil

	case ActivitiesLoadedMsg:
		a.detail.SetActivities(msg.Activities)
		return a, nil

	// --- Navigation messages ---

	case ViewSelectedMsg:
		a.currentView = msg.View
		if len(msg.View) > 0 {
			a.statusbar.context = strings.ToUpper(msg.View[:1]) + msg.View[1:]
		} else {
			a.statusbar.context = msg.View
		}
		return a, a.refreshCurrentView()

	case ProjectSelectedMsg:
		a.currentView = "project:" + msg.ID
		// Find the project title for the status bar.
		for _, p := range a.projects {
			if p.ID == msg.ID {
				a.statusbar.context = p.Title
				break
			}
		}
		return a, a.refreshCurrentView()

	case AreaSelectedMsg:
		a.currentView = "area:" + msg.ID
		// Find the area title for the status bar.
		for _, ar := range a.areas {
			if ar.ID == msg.ID {
				a.statusbar.context = ar.Title
				break
			}
		}
		return a, a.refreshCurrentView()

	// --- Task action messages from list pane ---

	case TaskSelectedMsg:
		id := msg.ID
		// Find the task in the list and populate the detail pane.
		for i := range a.list.tasks {
			if a.list.tasks[i].ID == id {
				a.detail.SetTask(&a.list.tasks[i])
				break
			}
		}
		a.focus = PaneDetail
		return a, a.refreshDetailByID(id)

	case TaskCompletedMsg:
		id := msg.ID
		return a, func() tea.Msg {
			if err := a.client.CompleteTask(context.Background(), id); err != nil {
				return ErrorMsg{Err: err}
			}
			return RefreshMsg{}
		}

	case TitleUpdatedMsg:
		id, title := msg.ID, msg.Title
		return a, func() tea.Msg {
			if err := a.client.UpdateTaskTitle(context.Background(), id, title); err != nil {
				return ErrorMsg{Err: err}
			}
			return RefreshMsg{}
		}

	case CreateTaskSignal:
		overlay := newInputOverlay("New task title:", func(title string) tea.Cmd {
			return func() tea.Msg {
				task, err := a.client.CreateTask(context.Background(), title)
				if err != nil {
					return ErrorMsg{Err: err}
				}
				return TaskCreatedMsg{Task: *task}
			}
		})
		a.inputOverlay = &overlay
		return a, nil

	case TaskCreatedMsg:
		return a, a.refreshCurrentView()

	case CancelTaskSignal:
		if t := a.list.SelectedTask(); t != nil {
			id := t.ID
			return a, func() tea.Msg {
				if err := a.client.CancelTask(context.Background(), id); err != nil {
					return ErrorMsg{Err: err}
				}
				return RefreshMsg{}
			}
		}
		return a, nil

	case DeleteTaskSignal:
		if t := a.list.SelectedTask(); t != nil {
			id := t.ID
			c := NewConfirm("Delete task? This cannot be undone.", a.width, func() tea.Cmd {
				return func() tea.Msg {
					if err := a.client.DeleteTask(context.Background(), id); err != nil {
						return ErrorMsg{Err: err}
					}
					return RefreshMsg{}
				}
			})
			a.confirm = &c
		}
		return a, nil

	case MoveTaskSignal:
		if t := a.list.SelectedTask(); t != nil {
			taskID := t.ID
			items := make([]PickerItem, 0, len(a.projects)+1)
			items = append(items, PickerItem{ID: "", Label: "(no project / inbox)"})
			for _, p := range a.projects {
				items = append(items, PickerItem{ID: p.ID, Label: p.Title})
			}
			p := NewPicker("Move to project", items, a.width, a.height, func(projectID string) tea.Cmd {
				return func() tea.Msg {
					var pid *string
					if projectID != "" {
						pid = &projectID
					}
					if err := a.client.MoveTaskToProject(context.Background(), taskID, pid); err != nil {
						return ErrorMsg{Err: err}
					}
					return RefreshMsg{}
				}
			})
			a.picker = &p
		}
		return a, nil

	case RefreshMsg:
		return a, a.refreshCurrentView()

	// --- Detail pane messages ---

	case EditNotesMsg:
		// v0: flash a status hint; full $EDITOR integration is deferred.
		a.statusbar.flash = "Edit notes via API not yet implemented"
		return a, nil

	case ToggleChecklistItemMsg:
		taskID, itemID, done := msg.TaskID, msg.ItemID, msg.Done
		return a, func() tea.Msg {
			var err error
			if done {
				err = a.client.CompleteChecklistItem(context.Background(), taskID, itemID)
			} else {
				err = a.client.UncompleteChecklistItem(context.Background(), taskID, itemID)
			}
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return detailRefreshMsg{taskID: taskID}
		}

	case AddChecklistItemMsg:
		taskID, title := msg.TaskID, msg.Title
		return a, func() tea.Msg {
			if _, err := a.client.AddChecklistItem(context.Background(), taskID, title); err != nil {
				return ErrorMsg{Err: err}
			}
			return detailRefreshMsg{taskID: taskID}
		}

	case DeleteChecklistItemMsg:
		taskID, itemID := msg.TaskID, msg.ItemID
		return a, func() tea.Msg {
			if err := a.client.DeleteChecklistItem(context.Background(), taskID, itemID); err != nil {
				return ErrorMsg{Err: err}
			}
			return detailRefreshMsg{taskID: taskID}
		}

	case AddCommentMsg:
		taskID, content := msg.TaskID, msg.Content
		return a, func() tea.Msg {
			if _, err := a.client.AddActivity(context.Background(), taskID, "human", "comment", content); err != nil {
				return ErrorMsg{Err: err}
			}
			return detailRefreshMsg{taskID: taskID}
		}

	case detailRefreshMsg:
		return a, a.refreshDetailByID(msg.taskID)

	// --- Search overlay messages ---

	case SearchQueryMsg:
		a.list.SetFilter(msg.Query)
		return a, nil

	// --- SSE messages ---

	case SSEEventMsg:
		// Trigger a targeted refresh based on event type, then keep listening.
		refreshCmd := a.handleSSEEvent(msg.Event)
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

	case ScheduleTaskSignal:
		if t := a.list.SelectedTask(); t != nil {
			s := NewSchedule(t.ID, a.width)
			a.schedule = &s
		}
		return a, nil

	case ScheduleSelectedMsg:
		taskID := msg.TaskID
		schedule := msg.Schedule
		return a, func() tea.Msg {
			err := a.client.UpdateTaskSchedule(context.Background(), taskID, schedule)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return ScheduleUpdatedMsg{ID: taskID, Schedule: schedule}
		}

	case ScheduleUpdatedMsg:
		return a, a.refreshCurrentView()
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
	if a.schedule != nil {
		return tea.NewView(a.renderOverlay(full, a.schedule.View()))
	}
	if a.picker != nil {
		return tea.NewView(a.renderOverlay(full, a.picker.View()))
	}
	if a.inputOverlay != nil {
		return tea.NewView(a.renderOverlay(full, a.inputOverlay.View()))
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
			baseRunes := []rune(lines[row])
			overlayRunes := []rune(padding + ol)
			if leftPad+len([]rune(ol)) <= len(baseRunes) {
				merged := string(baseRunes[:leftPad]) + string(overlayRunes[leftPad:]) + string(baseRunes[leftPad+len([]rune(ol)):])
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
	a.list.SetSize(listWidth, contentHeight)
	a.detail.SetSize(detailWidth, contentHeight)
	a.statusbar.width = a.width

	return a
}

// newPalette builds a Palette pre-populated with context-aware commands.
func (a App) newPalette() Palette {
	cmds := []Command{
		{
			Name:     "Go to Inbox",
			Category: "Navigation",
			Action: func() tea.Cmd {
				return func() tea.Msg { return ViewSelectedMsg{View: "inbox"} }
			},
		},
		{
			Name:     "Go to Today",
			Category: "Navigation",
			Action: func() tea.Cmd {
				return func() tea.Msg { return ViewSelectedMsg{View: "today"} }
			},
		},
		{
			Name:     "Go to Upcoming",
			Category: "Navigation",
			Action: func() tea.Cmd {
				return func() tea.Msg { return ViewSelectedMsg{View: "upcoming"} }
			},
		},
		{
			Name:     "Go to Someday",
			Category: "Navigation",
			Action: func() tea.Cmd {
				return func() tea.Msg { return ViewSelectedMsg{View: "someday"} }
			},
		},
		{
			Name:     "Refresh",
			Category: "System",
			Action: func() tea.Cmd {
				return a.cmdLoadInbox()
			},
		},
		{
			Name:     "Quit",
			Category: "System",
			Action: func() tea.Cmd {
				return tea.Quit
			},
		},
	}
	return NewPalette(cmds, a.width, a.height)
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

func (a App) cmdLoadToday() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListToday(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdLoadUpcoming() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListUpcoming(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdLoadSomeday() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListSomeday(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdLoadLogbook() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListLogbook(context.Background())
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdLoadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListTasksByProject(context.Background(), projectID)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return TasksLoadedMsg{Tasks: tasks}
	}
}

func (a App) cmdLoadAreaTasks(areaID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.ListTasksByArea(context.Background(), areaID)
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

// handleSSEEvent maps an incoming domain event to the appropriate refresh command(s).
func (a *App) handleSSEEvent(evt client.DomainEvent) tea.Cmd {
	prefix := strings.Split(evt.Type, ".")[0]
	switch prefix {
	case "task":
		return a.refreshCurrentView()
	case "project":
		return tea.Batch(a.cmdLoadProjects(), a.refreshCurrentView())
	case "area":
		return tea.Batch(a.cmdLoadAreas(), a.refreshCurrentView())
	case "checklist", "activity":
		// Refresh the detail pane if a task is currently selected.
		if a.list.SelectedTask() != nil {
			return a.refreshDetail()
		}
	case "tag":
		return a.cmdLoadTags()
	case "location":
		return a.cmdLoadLocations()
	}
	return nil
}

// refreshCurrentView reloads the task list for whichever view is active.
func (a App) refreshCurrentView() tea.Cmd {
	switch {
	case a.currentView == "" || a.currentView == "inbox":
		return a.cmdLoadInbox()
	case a.currentView == "today":
		return a.cmdLoadToday()
	case a.currentView == "upcoming":
		return a.cmdLoadUpcoming()
	case a.currentView == "someday":
		return a.cmdLoadSomeday()
	case a.currentView == "logbook":
		return a.cmdLoadLogbook()
	case strings.HasPrefix(a.currentView, "project:"):
		id := strings.TrimPrefix(a.currentView, "project:")
		return a.cmdLoadProjectTasks(id)
	case strings.HasPrefix(a.currentView, "area:"):
		id := strings.TrimPrefix(a.currentView, "area:")
		return a.cmdLoadAreaTasks(id)
	default:
		return a.cmdLoadInbox()
	}
}

// detailRefreshMsg is an internal message to reload detail pane data for a
// specific task ID.
type detailRefreshMsg struct {
	taskID string
}

// refreshDetail reloads the checklist and activity log for the currently
// selected task (determined from the list pane).
func (a App) refreshDetail() tea.Cmd {
	task := a.list.SelectedTask()
	if task == nil {
		return nil
	}
	return a.refreshDetailByID(task.ID)
}

// refreshDetailByID reloads checklist and activities for a specific task ID.
func (a App) refreshDetailByID(id string) tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			items, err := a.client.ListChecklistItems(context.Background(), id)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return ChecklistLoadedMsg{TaskID: id, Items: items}
		},
		func() tea.Msg {
			acts, err := a.client.ListActivities(context.Background(), id)
			if err != nil {
				return ErrorMsg{Err: err}
			}
			return ActivitiesLoadedMsg{TaskID: id, Activities: acts}
		},
	)
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
