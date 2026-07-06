package api

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// RoutesDeps carries everything RegisterRoutes needs to build the domain HTTP
// handlers. This is intentionally minimal for Phase 1 Task 10: it delegates to
// the existing NewRouter/http.ServeMux stack. Task 11 restructures routing to
// register handlers directly on PocketBase's router and swaps the legacy JWT
// AuthService for the AuthProvider — AuthProvider is threaded through here now
// so the wiring in main.go is already forward-compatible.
type RoutesDeps struct {
	DB            *store.DB
	AuthProvider  auth.AuthProvider // reserved for Task 11 (not yet used by routing)
	AuthService   *service.AuthService
	EventStore    *event.EventStore
	Bus           *event.Bus
	StreamManager *event.StreamManager

	TaskSvc      *service.TaskService
	ProjectSvc   *service.ProjectService
	AreaSvc      *service.AreaService
	SectionSvc   *service.SectionService
	TagSvc       *service.TagService
	LocationSvc  *service.LocationService
	ChecklistSvc *service.ChecklistService
	ActivitySvc  *service.ActivityService
}

// RegisterRoutes builds the domain HTTP handler (via the existing NewRouter) and
// mounts it on PocketBase's router as a catch-all. PocketBase owns the /api/* and
// /_/* prefixes; our domain routes (/health, /auth/*, /tasks, /areas, …) live under
// a root "/{path...}" wildcard which Go's ServeMux resolves with lower precedence
// than PocketBase's more-specific patterns, so the two coexist without conflict.
func RegisterRoutes(se *core.ServeEvent, deps RoutesDeps) {
	authHandler := NewAuthHandler(deps.AuthService)
	areaHandler := NewAreaHandler(deps.AreaSvc)
	taskHandler := NewTaskHandler(deps.TaskSvc, deps.ProjectSvc, deps.SectionSvc, deps.AreaSvc)
	projectHandler := NewProjectHandler(deps.ProjectSvc, deps.AreaSvc)
	sectionHandler := NewSectionHandler(deps.SectionSvc)
	tagHandler := NewTagHandler(deps.TagSvc)
	locationHandler := NewLocationHandler(deps.LocationSvc)
	checklistHandler := NewChecklistHandler(deps.ChecklistSvc)
	activityHandler := NewActivityHandler(deps.ActivitySvc)
	viewHandler := NewViewHandler(deps.DB)
	eventsHandler := NewEventsHandler(deps.StreamManager)
	syncHandler := NewSyncHandler(deps.EventStore)

	handler := NewRouter(
		areaHandler,
		taskHandler,
		projectHandler,
		sectionHandler,
		tagHandler,
		locationHandler,
		checklistHandler,
		activityHandler,
		viewHandler,
		eventsHandler,
		syncHandler,
		authHandler,
		deps.AuthService,
	)

	se.Router.Any("/{path...}", func(e *core.RequestEvent) error {
		handler.ServeHTTP(e.Response, e.Request)
		return nil
	})
}
