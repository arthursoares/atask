package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// TaskHandler holds the TaskService and handles task HTTP routes.
type TaskHandler struct {
	tasks *service.TaskService
}

// NewTaskHandler constructs a TaskHandler.
func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
}

// RegisterRoutes registers all task routes on the mux.
func (h *TaskHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tasks", h.Create)
	mux.HandleFunc("GET /tasks", h.List)
	mux.HandleFunc("GET /tasks/{id}", h.Get)
	mux.HandleFunc("DELETE /tasks/{id}", h.Delete)
	mux.HandleFunc("POST /tasks/{id}/complete", h.Complete)
	mux.HandleFunc("POST /tasks/{id}/cancel", h.Cancel)
	mux.HandleFunc("PUT /tasks/{id}/title", h.UpdateTitle)
	mux.HandleFunc("PUT /tasks/{id}/notes", h.UpdateNotes)
	mux.HandleFunc("PUT /tasks/{id}/schedule", h.UpdateSchedule)
	mux.HandleFunc("PUT /tasks/{id}/start-date", h.SetStartDate)
	mux.HandleFunc("PUT /tasks/{id}/deadline", h.SetDeadline)
	mux.HandleFunc("PUT /tasks/{id}/project", h.MoveToProject)
	mux.HandleFunc("PUT /tasks/{id}/section", h.MoveToSection)
	mux.HandleFunc("PUT /tasks/{id}/area", h.MoveToArea)
	mux.HandleFunc("PUT /tasks/{id}/location", h.SetLocation)
	mux.HandleFunc("PUT /tasks/{id}/recurrence", h.SetRecurrence)
	mux.HandleFunc("POST /tasks/{id}/tags/{tagId}", h.AddTag)
	mux.HandleFunc("DELETE /tasks/{id}/tags/{tagId}", h.RemoveTag)
	mux.HandleFunc("POST /tasks/{id}/links/{taskId}", h.AddLink)
	mux.HandleFunc("DELETE /tasks/{id}/links/{taskId}", h.RemoveLink)
	mux.HandleFunc("PUT /tasks/{id}/reorder", h.Reorder)
	mux.HandleFunc("PUT /tasks/{id}/today-index", h.SetTodayIndex)
	mux.HandleFunc("POST /tasks/{id}/reopen", h.Reopen)
	mux.HandleFunc("PUT /tasks/{id}/time-slot", h.SetTimeSlot)
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	task, err := h.tasks.Create(r.Context(), body.Title, actorFromRequest(r))
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.TaskCreated), task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var (
		tasks []*domain.Task
		err   error
	)

	switch {
	case q.Get("project_id") != "":
		tasks, err = h.tasks.ListByProject(r.Context(), q.Get("project_id"))
	case q.Get("area_id") != "":
		tasks, err = h.tasks.ListByArea(r.Context(), q.Get("area_id"))
	case q.Get("section_id") != "":
		tasks, err = h.tasks.ListBySection(r.Context(), q.Get("section_id"))
	case q.Get("location_id") != "":
		tasks, err = h.tasks.ListByLocation(r.Context(), q.Get("location_id"))
	case q.Get("schedule") != "":
		schedule, parseErr := domain.ParseSchedule(q.Get("schedule"))
		if parseErr != nil {
			RespondError(w, http.StatusBadRequest, parseErr.Error())
			return
		}
		tasks, err = h.tasks.ListBySchedule(r.Context(), schedule)
	default:
		tasks, err = h.tasks.List(r.Context())
	}

	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter by status (default: pending only)
	statusFilter := q.Get("status")
	if statusFilter == "" {
		statusFilter = "pending"
	}
	if statusFilter != "all" {
		var filtered []*domain.Task
		for _, t := range tasks {
			switch statusFilter {
			case "pending":
				if t.Status == domain.StatusPending {
					filtered = append(filtered, t)
				}
			case "completed":
				if t.Status == domain.StatusCompleted {
					filtered = append(filtered, t)
				}
			case "cancelled":
				if t.Status == domain.StatusCancelled {
					filtered = append(filtered, t)
				}
			}
		}
		tasks = filtered
	}

	RespondJSON(w, http.StatusOK, tasks)
}

func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := h.tasks.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.tasks.Delete(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskDeleted), map[string]string{"id": id})
}

func (h *TaskHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.tasks.Complete(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskCompleted), map[string]string{"id": id})
}

func (h *TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.tasks.Cancel(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskCancelled), map[string]string{"id": id})
}

func (h *TaskHandler) UpdateTitle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.UpdateTitle(r.Context(), id, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskTitleChanged), map[string]string{"id": id})
}

func (h *TaskHandler) UpdateNotes(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Notes string `json:"notes"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.UpdateNotes(r.Context(), id, body.Notes, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskNotesChanged), map[string]string{"id": id})
}

func (h *TaskHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Schedule string `json:"schedule"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	schedule, err := domain.ParseSchedule(body.Schedule)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.tasks.UpdateSchedule(r.Context(), id, schedule, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskScheduledToday), map[string]string{"id": id})
}

func (h *TaskHandler) SetStartDate(w http.ResponseWriter, r *http.Request) {
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

	if err := h.tasks.SetStartDate(r.Context(), id, date, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskStartDateSet), map[string]string{"id": id})
}

func (h *TaskHandler) SetDeadline(w http.ResponseWriter, r *http.Request) {
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

	if err := h.tasks.SetDeadline(r.Context(), id, date, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskDeadlineSet), map[string]string{"id": id})
}

func (h *TaskHandler) MoveToProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.MoveToProject(r.Context(), id, body.ID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskMovedToProject), map[string]string{"id": id})
}

func (h *TaskHandler) MoveToSection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.MoveToSection(r.Context(), id, body.ID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskMovedToSection), map[string]string{"id": id})
}

func (h *TaskHandler) MoveToArea(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.MoveToArea(r.Context(), id, body.ID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskMovedToArea), map[string]string{"id": id})
}

func (h *TaskHandler) SetLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		ID *string `json:"id"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.SetLocation(r.Context(), id, body.ID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskLocationSet), map[string]string{"id": id})
}

func (h *TaskHandler) SetRecurrence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body *domain.RecurrenceRule
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.SetRecurrence(r.Context(), id, body, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskRecurrenceSet), map[string]string{"id": id})
}

func (h *TaskHandler) AddTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tagID := r.PathValue("tagId")

	if err := h.tasks.AddTag(r.Context(), id, tagID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskTagAdded), map[string]string{"id": id, "tag_id": tagID})
}

func (h *TaskHandler) RemoveTag(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tagID := r.PathValue("tagId")

	if err := h.tasks.RemoveTag(r.Context(), id, tagID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskTagRemoved), map[string]string{"id": id, "tag_id": tagID})
}

func (h *TaskHandler) AddLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	taskID := r.PathValue("taskId")

	if err := h.tasks.AddLink(r.Context(), id, taskID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskLinkAdded), map[string]string{"id": id, "related_task_id": taskID})
}

func (h *TaskHandler) RemoveLink(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	taskID := r.PathValue("taskId")

	if err := h.tasks.RemoveLink(r.Context(), id, taskID, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskLinkRemoved), map[string]string{"id": id, "related_task_id": taskID})
}

func (h *TaskHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Index int `json:"index"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.Reorder(r.Context(), id, body.Index, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskReordered), map[string]string{"id": id})
}

func (h *TaskHandler) SetTodayIndex(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Index *int `json:"index"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.SetTodayIndex(r.Context(), id, body.Index, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskTodayIndexSet), map[string]string{"id": id})
}

func (h *TaskHandler) Reopen(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.tasks.Reopen(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskReopened), map[string]string{"id": id})
}

func (h *TaskHandler) SetTimeSlot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		TimeSlot *string `json:"time_slot"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tasks.SetTimeSlot(r.Context(), id, body.TimeSlot, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "task not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TaskTimeSlotSet), map[string]string{"id": id})
}
