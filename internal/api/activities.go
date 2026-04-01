package api

import (
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// ActivityHandler holds the ActivityService and handles activity HTTP routes.
type ActivityHandler struct {
	activities *service.ActivityService
}

// NewActivityHandler constructs an ActivityHandler.
func NewActivityHandler(activities *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activities: activities}
}

// RegisterRoutes registers all activity routes on the mux.
func (h *ActivityHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks/{id}/activity", h.Add)
	mux.HandleFunc("GET /tasks/{id}/activity", h.ListByTask)
}

func (h *ActivityHandler) Add(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	var body struct {
		ActorType string `json:"actor_type"`
		Type      string `json:"type"`
		Content   string `json:"content"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	actorType := domain.ActorType(body.ActorType)
	activityType := domain.ActivityType(body.Type)

	activity, err := h.activities.Add(r.Context(), taskID, actorFromRequest(r), actorType, activityType, body.Content)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.ActivityAdded), activity)
}

func (h *ActivityHandler) ListByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	activities, err := h.activities.ListByTask(r.Context(), taskID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, activities)
}
