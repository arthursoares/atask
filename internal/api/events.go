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
	mux.Handle("GET /events/stream", h.stream)
}
