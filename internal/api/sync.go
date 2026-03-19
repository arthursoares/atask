package api

import (
	"net/http"
	"strconv"

	"github.com/atask/atask/internal/event"
)

// SyncHandler handles delta-sync endpoints.
type SyncHandler struct {
	events *event.EventStore
}

// NewSyncHandler constructs a SyncHandler.
func NewSyncHandler(events *event.EventStore) *SyncHandler {
	return &SyncHandler{events: events}
}

// RegisterRoutes registers sync routes on the mux.
func (h *SyncHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /sync/deltas", h.Deltas)
}

// Deltas returns all delta events with ID > since cursor.
func (h *SyncHandler) Deltas(w http.ResponseWriter, r *http.Request) {
	var cursor int64
	if s := r.URL.Query().Get("since"); s != "" {
		var err error
		cursor, err = strconv.ParseInt(s, 10, 64)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "invalid since parameter")
			return
		}
	}

	deltas, err := h.events.DeltasSince(r.Context(), cursor)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, deltas)
}
