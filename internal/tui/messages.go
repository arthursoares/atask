package tui

import "github.com/atask/atask/internal/client"

// Data loaded from API.
type (
	TasksLoadedMsg      struct{ Tasks []client.Task }
	AreasLoadedMsg      struct{ Areas []client.Area }
	ProjectsLoadedMsg   struct{ Projects []client.Project }
	TagsLoadedMsg       struct{ Tags []client.Tag }
	LocationsLoadedMsg  struct{ Locations []client.Location }
	SectionsLoadedMsg   struct{ Sections []client.Section }
	ChecklistLoadedMsg  struct{ Items []client.ChecklistItem }
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
