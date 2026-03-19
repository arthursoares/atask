package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// SectionHandler holds the SectionService and handles section HTTP routes.
type SectionHandler struct {
	sections *service.SectionService
}

// NewSectionHandler constructs a SectionHandler.
func NewSectionHandler(sections *service.SectionService) *SectionHandler {
	return &SectionHandler{sections: sections}
}

// RegisterRoutes registers all section routes on the mux.
func (h *SectionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /projects/{id}/sections", h.Create)
	mux.HandleFunc("GET /projects/{id}/sections", h.ListByProject)
	mux.HandleFunc("PUT /projects/{id}/sections/{sid}", h.Rename)
	mux.HandleFunc("DELETE /projects/{id}/sections/{sid}", h.Delete)
}

func (h *SectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	section, err := h.sections.Create(r.Context(), body.Title, projectID, actorFromRequest(r))
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.SectionCreated), section)
}

func (h *SectionHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")

	sections, err := h.sections.ListByProject(r.Context(), projectID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, sections)
}

func (h *SectionHandler) Rename(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("sid")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.sections.Rename(r.Context(), sid, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "section not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.SectionRenamed), map[string]string{"id": sid})
}

func (h *SectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("sid")
	cascade := r.URL.Query().Get("cascade") == "true"

	if err := h.sections.Delete(r.Context(), sid, actorFromRequest(r), cascade); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "section not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.SectionDeleted), map[string]string{"id": sid})
}
