package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// ProjectHandler holds the ProjectService and handles project HTTP routes.
type ProjectHandler struct {
	projects *service.ProjectService
}

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(projects *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// RegisterRoutes registers all project routes on the mux.
func (h *ProjectHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /projects", h.Create)
	mux.HandleFunc("GET /projects", h.List)
	mux.HandleFunc("GET /projects/{id}", h.Get)
	mux.HandleFunc("DELETE /projects/{id}", h.Delete)
	mux.HandleFunc("POST /projects/{id}/complete", h.Complete)
	mux.HandleFunc("POST /projects/{id}/cancel", h.Cancel)
	mux.HandleFunc("PUT /projects/{id}/title", h.UpdateTitle)
	mux.HandleFunc("PUT /projects/{id}/notes", h.UpdateNotes)
	mux.HandleFunc("PUT /projects/{id}/deadline", h.SetDeadline)
	mux.HandleFunc("PUT /projects/{id}/area", h.MoveToArea)
	mux.HandleFunc("POST /projects/{id}/tags/{tagId}", h.AddTag)
	mux.HandleFunc("DELETE /projects/{id}/tags/{tagId}", h.RemoveTag)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	project, err := h.projects.Create(r.Context(), body.Title, actorFromRequest(r))
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.ProjectCreated), project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	project, err := h.projects.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.projects.Delete(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectDeleted), map[string]string{"id": id})
}

func (h *ProjectHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.projects.Complete(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectCompleted), map[string]string{"id": id})
}

func (h *ProjectHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.projects.Cancel(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectCancelled), map[string]string{"id": id})
}

func (h *ProjectHandler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.projects.UpdateTitle(r.Context(), id, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectTitleChanged), map[string]string{"id": id})
}

func (h *ProjectHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Notes string `json:"notes"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.projects.UpdateNotes(r.Context(), id, body.Notes, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectNotesChanged), map[string]string{"id": id})
}

func (h *ProjectHandler) SetDeadline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Date *string `json:"date"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var date *time.Time
	if body.Date != nil {
		parsed, err := time.Parse("2006-01-02", *body.Date)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "invalid date format, expected YYYY-MM-DD")
			return
		}
		date = &parsed
	}

	if err := h.projects.SetDeadline(r.Context(), id, date, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectDeadlineSet), map[string]string{"id": id})
}

func (h *ProjectHandler) MoveToArea(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.projects.MoveToArea(r.Context(), id, body.ID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectMovedToArea), map[string]string{"id": id})
}

func (h *ProjectHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tagID := r.PathValue("tagId")

	if err := h.projects.AddTag(r.Context(), id, tagID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectTagAdded), map[string]string{"id": id, "tag_id": tagID})
}

func (h *ProjectHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tagID := r.PathValue("tagId")

	if err := h.projects.RemoveTag(r.Context(), id, tagID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectTagRemoved), map[string]string{"id": id, "tag_id": tagID})
}
