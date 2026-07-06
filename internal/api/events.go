package api

import (
	"net/http"

	"github.com/atask/atask/internal/event"
)

// EventsHandler wires the SSE StreamManager to an HTTP route.
type EventsHandler struct {
	stream *event.StreamManager
}

// NewEventsHandler constructs an EventsHandler.
func NewEventsHandler(stream *event.StreamManager) *EventsHandler {
	return &EventsHandler{stream: stream}
}

// RegisterRoutes registers the SSE stream route on the mux.
func (h *EventsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /events/stream", h.Stream)
}

// Stream extracts the authenticated user from the request context and
// delegates to the StreamManager, scoping delivered events to that user.
// Kept in internal/api (rather than internal/event) so the SSE stream can
// reuse UserIDFromContext without internal/event importing internal/api
// (which would create an import cycle, since internal/api already imports
// internal/event).
func (h *EventsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	h.stream.ServeHTTP(w, r, userID)
}
