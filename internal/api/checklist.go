package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// ChecklistHandler holds the ChecklistService and handles checklist HTTP routes.
type ChecklistHandler struct {
	checklist *service.ChecklistService
}

// NewChecklistHandler constructs a ChecklistHandler.
func NewChecklistHandler(checklist *service.ChecklistService) *ChecklistHandler {
	return &ChecklistHandler{checklist: checklist}
}

// RegisterRoutes registers all checklist routes on the mux.
func (h *ChecklistHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks/{id}/checklist", h.AddItem)
	mux.HandleFunc("GET /tasks/{id}/checklist", h.ListByTask)
	mux.HandleFunc("PUT /tasks/{id}/checklist/{itemId}", h.UpdateTitle)
	mux.HandleFunc("POST /tasks/{id}/checklist/{itemId}/complete", h.CompleteItem)
	mux.HandleFunc("POST /tasks/{id}/checklist/{itemId}/uncomplete", h.UncompleteItem)
	mux.HandleFunc("DELETE /tasks/{id}/checklist/{itemId}", h.RemoveItem)
}

func (h *ChecklistHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	item, err := h.checklist.AddItem(r.Context(), body.Title, taskID, actorFromRequest(r))
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.ChecklistItemAdded), item)
}

func (h *ChecklistHandler) ListByTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")

	items, err := h.checklist.ListByTask(r.Context(), taskID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, items)
}

func (h *ChecklistHandler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemId")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.checklist.UpdateTitle(r.Context(), itemID, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "checklist item not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ChecklistItemTitleChanged), map[string]string{"id": itemID})
}

func (h *ChecklistHandler) CompleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemId")

	if err := h.checklist.CompleteItem(r.Context(), itemID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "checklist item not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ChecklistItemCompleted), map[string]string{"id": itemID})
}

func (h *ChecklistHandler) UncompleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemId")

	if err := h.checklist.UncompleteItem(r.Context(), itemID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "checklist item not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ChecklistItemUncompleted), map[string]string{"id": itemID})
}

func (h *ChecklistHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("itemId")

	if err := h.checklist.RemoveItem(r.Context(), itemID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "checklist item not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ChecklistItemRemoved), map[string]string{"id": itemID})
}
