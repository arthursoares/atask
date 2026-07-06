package api

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// RoutesDeps carries everything RegisterRoutes needs to build the domain HTTP
// handlers and wire authentication.
//
//   - AuthProvider resolves Bearer tokens and user records (PocketBase-backed).
//   - AuthService still backs the legacy JWT /auth/login + /auth/register routes
//     (Task 12 replaces these with AuthProvider) and satisfies APIKeyValidator
//     for the ApiKey auth path.
type RoutesDeps struct {
	DB            *store.DB
	AuthProvider  auth.AuthProvider
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

// bridge adapts an http.Handler to PocketBase's *core.RequestEvent callback.
// PocketBase's RequestEvent embeds the underlying http.ResponseWriter and
// *http.Request, so we pass them through unchanged. Any middleware that mutates
// the request context (RequestID, auth) is composed into h before it reaches
// this bridge and is preserved on the underlying request.
//
// This is the single routing primitive in the codebase. Task 14's admin routes
// reuse it with requireAdmin substituted for requireAuth.
func bridge(h http.Handler) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		h.ServeHTTP(e.Response, e.Request)
		return nil
	}
}

// RegisterRoutes mounts every domain route directly on PocketBase's router with
// per-route authentication. Public routes (/health, /auth/login, /auth/register)
// bypass auth; everything else is wrapped with requireAuth. The transitional
// catch-all + NewRouter stack from Task 10 has been removed.
//
// PocketBase owns the /api/* and /_/* prefixes; our routes live under distinct
// paths (/health, /auth/*, /tasks, …), so the two route sets coexist without
// conflict.
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

	authMW := requireAuth(deps.AuthProvider, deps.AuthService)

	// common applies the request-scoped middleware that ran for every domain
	// route in the pre-Task-11 stack: RequestID (X-Request-ID header + context)
	// wrapping Logging (slog request line). RequestID is outermost so its ID is
	// available to the logger and downstream handlers.
	common := func(h http.Handler) http.Handler {
		return RequestID(Logging(h))
	}

	// public wraps a handler with the common middleware only (no auth).
	public := func(h http.HandlerFunc) func(*core.RequestEvent) error {
		return bridge(common(h))
	}

	// protect wraps a handler with the common middleware and requireAuth.
	protect := func(h http.HandlerFunc) func(*core.RequestEvent) error {
		return bridge(common(authMW(http.HandlerFunc(h))))
	}

	// --- Public routes ---
	se.Router.GET("/health", public(handleHealth))
	se.Router.POST("/auth/register", public(authHandler.Register))
	se.Router.POST("/auth/login", public(authHandler.Login))
	// NOTE: /auth/refresh and /auth/providers are added by Task 12 (handlers do
	// not exist yet).

	// --- Auth (protected) ---
	se.Router.GET("/auth/me", protect(authHandler.GetMe))
	se.Router.PUT("/auth/me", protect(authHandler.UpdateMe))
	se.Router.GET("/auth/api-keys", protect(authHandler.ListAPIKeys))
	se.Router.POST("/auth/api-keys", protect(authHandler.CreateAPIKey))
	se.Router.PUT("/auth/api-keys/{id}", protect(authHandler.UpdateAPIKey))
	se.Router.DELETE("/auth/api-keys/{id}", protect(authHandler.DeleteAPIKey))

	// --- Tasks ---
	se.Router.POST("/tasks", protect(taskHandler.Create))
	se.Router.GET("/tasks", protect(taskHandler.List))
	se.Router.GET("/tasks/{id}", protect(taskHandler.Get))
	se.Router.DELETE("/tasks/{id}", protect(taskHandler.Delete))
	se.Router.POST("/tasks/{id}/complete", protect(taskHandler.Complete))
	se.Router.POST("/tasks/{id}/cancel", protect(taskHandler.Cancel))
	se.Router.PUT("/tasks/{id}/title", protect(taskHandler.UpdateTitle))
	se.Router.PUT("/tasks/{id}/notes", protect(taskHandler.UpdateNotes))
	se.Router.PUT("/tasks/{id}/schedule", protect(taskHandler.UpdateSchedule))
	se.Router.PUT("/tasks/{id}/start-date", protect(taskHandler.SetStartDate))
	se.Router.PUT("/tasks/{id}/deadline", protect(taskHandler.SetDeadline))
	se.Router.PUT("/tasks/{id}/project", protect(taskHandler.MoveToProject))
	se.Router.PUT("/tasks/{id}/section", protect(taskHandler.MoveToSection))
	se.Router.PUT("/tasks/{id}/area", protect(taskHandler.MoveToArea))
	se.Router.PUT("/tasks/{id}/location", protect(taskHandler.SetLocation))
	se.Router.PUT("/tasks/{id}/recurrence", protect(taskHandler.SetRecurrence))
	se.Router.POST("/tasks/{id}/tags/{tagId}", protect(taskHandler.AddTag))
	se.Router.DELETE("/tasks/{id}/tags/{tagId}", protect(taskHandler.RemoveTag))
	se.Router.POST("/tasks/{id}/links/{taskId}", protect(taskHandler.AddLink))
	se.Router.DELETE("/tasks/{id}/links/{taskId}", protect(taskHandler.RemoveLink))
	se.Router.PUT("/tasks/{id}/reorder", protect(taskHandler.Reorder))
	se.Router.PUT("/tasks/{id}/today-index", protect(taskHandler.SetTodayIndex))
	se.Router.POST("/tasks/{id}/reopen", protect(taskHandler.Reopen))
	se.Router.PATCH("/tasks/{id}", protect(taskHandler.Patch))

	// --- Projects ---
	se.Router.POST("/projects", protect(projectHandler.Create))
	se.Router.GET("/projects", protect(projectHandler.List))
	se.Router.GET("/projects/{id}", protect(projectHandler.Get))
	se.Router.DELETE("/projects/{id}", protect(projectHandler.Delete))
	se.Router.POST("/projects/{id}/complete", protect(projectHandler.Complete))
	se.Router.POST("/projects/{id}/cancel", protect(projectHandler.Cancel))
	se.Router.PUT("/projects/{id}/title", protect(projectHandler.UpdateTitle))
	se.Router.PUT("/projects/{id}/notes", protect(projectHandler.UpdateNotes))
	se.Router.PUT("/projects/{id}/deadline", protect(projectHandler.SetDeadline))
	se.Router.PUT("/projects/{id}/area", protect(projectHandler.MoveToArea))
	se.Router.PUT("/projects/{id}/color", protect(projectHandler.UpdateColor))
	se.Router.POST("/projects/{id}/tags/{tagId}", protect(projectHandler.AddTag))
	se.Router.DELETE("/projects/{id}/tags/{tagId}", protect(projectHandler.RemoveTag))
	se.Router.PATCH("/projects/{id}", protect(projectHandler.Patch))

	// --- Areas ---
	se.Router.POST("/areas", protect(areaHandler.Create))
	se.Router.GET("/areas", protect(areaHandler.List))
	se.Router.GET("/areas/{id}", protect(areaHandler.Get))
	se.Router.PUT("/areas/{id}", protect(areaHandler.Rename))
	se.Router.DELETE("/areas/{id}", protect(areaHandler.Delete))
	se.Router.POST("/areas/{id}/archive", protect(areaHandler.Archive))
	se.Router.POST("/areas/{id}/unarchive", protect(areaHandler.Unarchive))
	se.Router.PATCH("/areas/{id}", protect(areaHandler.Patch))

	// --- Sections ---
	se.Router.POST("/projects/{id}/sections", protect(sectionHandler.Create))
	se.Router.GET("/projects/{id}/sections", protect(sectionHandler.ListByProject))
	se.Router.PUT("/projects/{id}/sections/{sid}", protect(sectionHandler.Rename))
	se.Router.PUT("/projects/{id}/sections/{sid}/reorder", protect(sectionHandler.Reorder))
	se.Router.DELETE("/projects/{id}/sections/{sid}", protect(sectionHandler.Delete))

	// --- Tags ---
	se.Router.POST("/tags", protect(tagHandler.Create))
	se.Router.GET("/tags", protect(tagHandler.List))
	se.Router.GET("/tags/{id}", protect(tagHandler.Get))
	se.Router.PUT("/tags/{id}", protect(tagHandler.Rename))
	se.Router.DELETE("/tags/{id}", protect(tagHandler.Delete))

	// --- Locations ---
	se.Router.POST("/locations", protect(locationHandler.Create))
	se.Router.GET("/locations", protect(locationHandler.List))
	se.Router.GET("/locations/{id}", protect(locationHandler.Get))
	se.Router.PUT("/locations/{id}", protect(locationHandler.Rename))
	se.Router.DELETE("/locations/{id}", protect(locationHandler.Delete))

	// --- Checklist ---
	se.Router.POST("/tasks/{id}/checklist", protect(checklistHandler.AddItem))
	se.Router.GET("/tasks/{id}/checklist", protect(checklistHandler.ListByTask))
	se.Router.PUT("/tasks/{id}/checklist/{itemId}", protect(checklistHandler.UpdateTitle))
	se.Router.POST("/tasks/{id}/checklist/{itemId}/complete", protect(checklistHandler.CompleteItem))
	se.Router.POST("/tasks/{id}/checklist/{itemId}/uncomplete", protect(checklistHandler.UncompleteItem))
	se.Router.DELETE("/tasks/{id}/checklist/{itemId}", protect(checklistHandler.RemoveItem))
	se.Router.PUT("/tasks/{id}/checklist/{itemId}/reorder", protect(checklistHandler.ReorderItem))

	// --- Activities ---
	se.Router.POST("/tasks/{id}/activity", protect(activityHandler.Add))
	se.Router.GET("/tasks/{id}/activity", protect(activityHandler.ListByTask))

	// --- Views ---
	se.Router.GET("/views/inbox", protect(viewHandler.Inbox))
	se.Router.GET("/views/today", protect(viewHandler.Today))
	se.Router.GET("/views/upcoming", protect(viewHandler.Upcoming))
	se.Router.GET("/views/someday", protect(viewHandler.Someday))
	se.Router.GET("/views/logbook", protect(viewHandler.Logbook))

	// --- Events (SSE) ---
	se.Router.GET("/events/stream", protect(eventsHandler.Stream))

	// --- Sync ---
	se.Router.GET("/sync/deltas", protect(syncHandler.Deltas))
}
