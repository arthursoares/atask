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
	areas    *service.AreaService
}

// NewProjectHandler constructs a ProjectHandler.
func NewProjectHandler(projects *service.ProjectService, areas *service.AreaService) *ProjectHandler {
	return &ProjectHandler{projects: projects, areas: areas}
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
	mux.HandleFunc("PUT /projects/{id}/color", h.UpdateColor)
	mux.HandleFunc("POST /projects/{id}/tags/{tagId}", h.AddTag)
	mux.HandleFunc("DELETE /projects/{id}/tags/{tagId}", h.RemoveTag)
	mux.HandleFunc("PATCH /projects/{id}", h.Patch)
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		ID    string `json:"id,omitempty"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	project, err := h.projects.Create(r.Context(), body.Title, actorFromRequest(r), body.ID)
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

	// Filter by status (default: pending only)
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "pending"
	}
	if statusFilter != "all" {
		var filtered []*domain.Project
		for _, p := range projects {
			switch statusFilter {
			case "pending":
				if p.Status == domain.StatusPending {
					filtered = append(filtered, p)
				}
			case "completed":
				if p.Status == domain.StatusCompleted {
					filtered = append(filtered, p)
				}
			case "cancelled":
				if p.Status == domain.StatusCancelled {
					filtered = append(filtered, p)
				}
			}
		}
		projects = filtered
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
		RespondDecodeError(w, err)
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
		RespondDecodeError(w, err)
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
		RespondDecodeError(w, err)
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

func (h *ProjectHandler) UpdateColor(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Color string `json:"color"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	if err := h.projects.UpdateColor(r.Context(), id, body.Color, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.ProjectColorChanged), map[string]string{"id": id})
}

func (h *ProjectHandler) MoveToArea(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
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

func (h *ProjectHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title    *string `json:"title"`
		Notes    *string `json:"notes"`
		Deadline *string `json:"deadline"`
		AreaID   *string `json:"areaId"`
		Color    *string `json:"color"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	actor := actorFromRequest(r)

	// --- Pre-validate: project exists ---
	if _, err := h.projects.Get(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "project not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// --- Pre-validate: parse date fields ---
	var deadline *time.Time
	if body.Deadline != nil && *body.Deadline != "" {
		t, err := time.Parse("2006-01-02", *body.Deadline)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "invalid deadline format")
			return
		}
		deadline = &t
	}

	// --- Pre-validate: referenced entities exist ---
	if body.AreaID != nil && *body.AreaID != "" {
		if _, err := h.areas.Get(r.Context(), *body.AreaID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				RespondError(w, http.StatusUnprocessableEntity, "area not found")
				return
			}
			RespondError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// --- Apply mutations (all pre-validations passed) ---
	if body.Title != nil {
		if err := h.projects.UpdateTitle(r.Context(), id, *body.Title, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if body.Notes != nil {
		if err := h.projects.UpdateNotes(r.Context(), id, *body.Notes, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if body.Deadline != nil {
		if err := h.projects.SetDeadline(r.Context(), id, deadline, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if body.AreaID != nil {
		aid := body.AreaID
		if *aid == "" {
			aid = nil
		}
		if err := h.projects.MoveToArea(r.Context(), id, aid, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}
	if body.Color != nil {
		if err := h.projects.UpdateColor(r.Context(), id, *body.Color, actor); err != nil {
			RespondError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	project, err := h.projects.Get(r.Context(), id)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, project)
}
