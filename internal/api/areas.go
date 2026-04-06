package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// AreaHandler holds the AreaService and handles area HTTP routes.
type AreaHandler struct {
	areas *service.AreaService
}

// NewAreaHandler constructs an AreaHandler.
func NewAreaHandler(areas *service.AreaService) *AreaHandler {
	return &AreaHandler{areas: areas}
}

// RegisterRoutes registers all area routes on the mux.
func (h *AreaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /areas", h.Create)
	mux.HandleFunc("GET /areas", h.List)
	mux.HandleFunc("GET /areas/{id}", h.Get)
	mux.HandleFunc("PUT /areas/{id}", h.Rename)
	mux.HandleFunc("DELETE /areas/{id}", h.Delete)
	mux.HandleFunc("POST /areas/{id}/archive", h.Archive)
	mux.HandleFunc("POST /areas/{id}/unarchive", h.Unarchive)
	mux.HandleFunc("PATCH /areas/{id}", h.Patch)
}

func (h *AreaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		ID    string `json:"id,omitempty"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	area, err := h.areas.Create(r.Context(), body.Title, actorFromRequest(r), body.ID)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.AreaCreated), area)
}

func (h *AreaHandler) List(w http.ResponseWriter, r *http.Request) {
	var areas []*domain.Area
	var err error

	if r.URL.Query().Get("include_archived") == "true" {
		areas, err = h.areas.ListAll(r.Context())
	} else {
		areas, err = h.areas.List(r.Context())
	}

	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, areas)
}

func (h *AreaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	area, err := h.areas.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "area not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, area)
}

func (h *AreaHandler) Rename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	if err := h.areas.Rename(r.Context(), id, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "area not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.AreaRenamed), map[string]string{"id": id})
}

func (h *AreaHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cascade := r.URL.Query().Get("cascade") == "true"

	if err := h.areas.Delete(r.Context(), id, actorFromRequest(r), cascade); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "area not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.AreaDeleted), map[string]string{"id": id})
}

func (h *AreaHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.areas.Archive(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "area not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.AreaArchived), map[string]string{"id": id})
}

func (h *AreaHandler) Unarchive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.areas.Unarchive(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "area not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.AreaUnarchived), map[string]string{"id": id})
}

func (h *AreaHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title *string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	actor := actorFromRequest(r)

	if body.Title != nil {
		if err := h.areas.Rename(r.Context(), id, *body.Title, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	area, err := h.areas.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusNotFound, "area not found")
		return
	}
	RespondJSON(w, http.StatusOK, area)
}
