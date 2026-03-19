package event

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"strings"
	"time"

	"github.com/atask/atask/internal/domain"
)

// StreamManager is an SSE HTTP handler with topic-based filtering backed by the event Bus.
type StreamManager struct {
	bus *Bus
}

// NewStreamManager creates a new StreamManager backed by the given Bus.
func NewStreamManager(bus *Bus) *StreamManager {
	return &StreamManager{bus: bus}
}

// ServeHTTP implements http.Handler — SSE endpoint with topic filtering.
func (sm *StreamManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Parse topics query param (comma-separated, default "*")
	topicsParam := r.URL.Query().Get("topics")
	var topics []string
	if topicsParam == "" {
		topics = []string{"*"}
	} else {
		for t := range strings.SplitSeq(topicsParam, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				topics = append(topics, t)
			}
		}
		if len(topics) == 0 {
			topics = []string{"*"}
		}
	}

	// Disable write deadline for this long-lived SSE connection.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		slog.Warn("SSE: failed to clear write deadline", "error", err)
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Buffered channel for incoming events
	ch := make(chan *domain.DomainEvent, 64)

	// Subscribe to each topic and track IDs for cleanup
	subIDs := make([]int, 0, len(topics))
	for _, topic := range topics {
		id := sm.bus.Subscribe(topic, func(e *domain.DomainEvent) {
			select {
			case ch <- e:
			default:
				slog.Warn("SSE event dropped: channel full", "event_type", e.Type, "event_id", e.ID)
			}
		})
		subIDs = append(subIDs, id)
	}

	// Unsubscribe on exit
	defer func() {
		for _, id := range subIDs {
			sm.bus.Unsubscribe(id)
		}
	}()

	// Flush headers to the client
	flusher.Flush()

	// Event loop
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-ch:
			// Merge standard event fields into payload so SSE clients
			// always know which entity the event refers to.
			data := make(map[string]any, len(e.Payload)+3)
			data["entity_type"] = e.EntityType
			data["entity_id"] = e.EntityID
			data["actor_id"] = e.ActorID
			maps.Copy(data, e.Payload)
			payload, err := json.Marshal(data)
			if err != nil {
				slog.Warn("SSE failed to marshal event payload", "error", err)
				payload = []byte("{}")
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\nid: %d\n\n", e.Type, payload, e.ID); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
