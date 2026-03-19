package tui

import "github.com/atask/atask/internal/client"

// --- Data loading messages ---

// TasksLoadedMsg is sent when a list of tasks has been fetched from the API.
type TasksLoadedMsg struct {
	Tasks []client.Task
}

// TaskCreatedMsg is sent when a new task has been created.
type TaskCreatedMsg struct {
	Task client.Task
}

// TaskCompletedMsg is sent when a task has been marked complete.
type TaskCompletedMsg struct {
	ID string
}

// TaskDeletedMsg is sent when a task has been deleted.
type TaskDeletedMsg struct {
	ID string
}

// ErrorMsg is sent when an operation fails.
type ErrorMsg struct {
	Err error
}

// --- SSE messages ---

// SSEEventMsg is sent when a domain event arrives over the SSE stream.
type SSEEventMsg struct {
	Event client.DomainEvent
}

// SSEDisconnectedMsg is sent when the SSE stream disconnects.
type SSEDisconnectedMsg struct{}

// --- Navigation messages ---

// ViewSelectedMsg is sent when the user selects a top-level view (e.g. "inbox", "today").
type ViewSelectedMsg struct {
	View string
}

// ProjectSelectedMsg is sent when the user selects a project in the sidebar.
type ProjectSelectedMsg struct {
	ID string
}

// AreaSelectedMsg is sent when the user selects an area in the sidebar.
type AreaSelectedMsg struct {
	ID string
}

// TagSelectedMsg is sent when the user selects a tag filter in the sidebar.
type TagSelectedMsg struct {
	ID string
}

// TaskSelectedMsg is sent when the user selects a task in the task list.
type TaskSelectedMsg struct {
	ID string
}

// --- UI control messages ---

// FocusPaneMsg is sent to shift keyboard focus to a specific pane index.
type FocusPaneMsg struct {
	Pane int
}

// FlashMsg is sent to display a transient status message in the status bar.
type FlashMsg struct {
	Message string
}

// RefreshMsg is sent to trigger a data refresh of the current view.
type RefreshMsg struct{}

// ClearFlashMsg is sent to clear the current flash/status message.
type ClearFlashMsg struct{}

// --- Collection loaded messages ---

// AreasLoadedMsg is sent when the list of areas has been fetched.
type AreasLoadedMsg struct {
	Areas []client.Area
}

// ProjectsLoadedMsg is sent when the list of projects has been fetched.
type ProjectsLoadedMsg struct {
	Projects []client.Project
}

// TagsLoadedMsg is sent when the list of tags has been fetched.
type TagsLoadedMsg struct {
	Tags []client.Tag
}

// LocationsLoadedMsg is sent when the list of locations has been fetched.
type LocationsLoadedMsg struct {
	Locations []client.Location
}

// SectionsLoadedMsg is sent when the list of sections for a project has been fetched.
type SectionsLoadedMsg struct {
	ProjectID string
	Sections  []client.Section
}

// ChecklistLoadedMsg is sent when the checklist items for a task have been fetched.
type ChecklistLoadedMsg struct {
	TaskID string
	Items  []client.ChecklistItem
}

// ActivitiesLoadedMsg is sent when the activity log for a task has been fetched.
type ActivitiesLoadedMsg struct {
	TaskID     string
	Activities []client.Activity
}

// --- Mutation result messages ---

// AreaCreatedMsg is sent when a new area has been created.
type AreaCreatedMsg struct {
	Area client.Area
}

// ProjectCreatedMsg is sent when a new project has been created.
type ProjectCreatedMsg struct {
	Project client.Project
}

// TagCreatedMsg is sent when a new tag has been created.
type TagCreatedMsg struct {
	Tag client.Tag
}

// TitleUpdatedMsg is sent when a task or project title has been updated.
type TitleUpdatedMsg struct {
	ID    string
	Title string
}

// ScheduleUpdatedMsg is sent when a task's schedule has been updated.
type ScheduleUpdatedMsg struct {
	ID       string
	Schedule string
}
