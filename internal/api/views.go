package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
)

// ViewHandler serves computed views over tasks.
type ViewHandler struct {
	queries *sqlc.Queries
}

// NewViewHandler constructs a ViewHandler backed by the given DB.
func NewViewHandler(db *store.DB) *ViewHandler {
	return &ViewHandler{
		queries: sqlc.New(db.DB),
	}
}

// RegisterRoutes registers all view routes on the mux.
func (h *ViewHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /views/inbox", h.Inbox)
	mux.HandleFunc("GET /views/today", h.Today)
	mux.HandleFunc("GET /views/upcoming", h.Upcoming)
	mux.HandleFunc("GET /views/someday", h.Someday)
	mux.HandleFunc("GET /views/logbook", h.Logbook)
}

// viewTaskFromRow converts a sqlc.Task row to a domain.Task.
func viewTaskFromRow(row sqlc.Task) *domain.Task {
	t := &domain.Task{
		ID:       row.ID,
		Notes:    row.Notes,
		Status:   domain.Status(row.Status),
		Schedule: domain.Schedule(row.Schedule),
		Index:    int(row.Index),
		Timestamps: domain.Timestamps{
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		},
	}

	if row.Title.Valid {
		t.Title = row.Title.String
	}

	if row.StartDate.Valid {
		parsed, err := time.Parse("2006-01-02", row.StartDate.String)
		if err == nil {
			t.StartDate = &parsed
		}
	}

	if row.Deadline.Valid {
		parsed, err := time.Parse("2006-01-02", row.Deadline.String)
		if err == nil {
			t.Deadline = &parsed
		}
	}

	if row.CompletedAt.Valid {
		ca := row.CompletedAt.Time
		t.CompletedAt = &ca
	}

	if row.TodayIndex.Valid {
		ti := int(row.TodayIndex.Int64)
		t.TodayIndex = &ti
	}

	if row.ProjectID.Valid {
		pid := row.ProjectID.String
		t.ProjectID = &pid
	}

	if row.SectionID.Valid {
		sid := row.SectionID.String
		t.SectionID = &sid
	}

	if row.AreaID.Valid {
		aid := row.AreaID.String
		t.AreaID = &aid
	}

	if row.LocationID.Valid {
		lid := row.LocationID.String
		t.LocationID = &lid
	}

	if row.RecurrenceRule.Valid {
		var rule domain.RecurrenceRule
		if err := json.Unmarshal([]byte(row.RecurrenceRule.String), &rule); err == nil {
			t.RecurrenceRule = &rule
		}
	}

	if row.Deleted != 0 && row.DeletedAt.Valid {
		da := row.DeletedAt.Time
		t.SoftDelete = domain.SoftDelete{
			Deleted:   true,
			DeletedAt: &da,
		}
	}

	return t
}

// Inbox handles GET /views/inbox — returns tasks with schedule=0, status=0, not deleted.
func (h *ViewHandler) Inbox(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ViewInbox(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = viewTaskFromRow(row)
	}
	RespondJSON(w, http.StatusOK, tasks)
}

// Today handles GET /views/today — returns tasks with schedule=1, status=0, start_date IS NULL OR <= today.
func (h *ViewHandler) Today(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	rows, err := h.queries.ViewToday(r.Context(), sql.NullString{String: today, Valid: true})
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = viewTaskFromRow(row)
	}
	RespondJSON(w, http.StatusOK, tasks)
}

// Upcoming handles GET /views/upcoming — returns tasks with start_date > today.
func (h *ViewHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	today := time.Now().Format("2006-01-02")
	rows, err := h.queries.ViewUpcoming(r.Context(), sql.NullString{String: today, Valid: true})
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = viewTaskFromRow(row)
	}
	RespondJSON(w, http.StatusOK, tasks)
}

// Someday handles GET /views/someday — returns tasks with schedule=2, status=0.
func (h *ViewHandler) Someday(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ViewSomeday(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = viewTaskFromRow(row)
	}
	RespondJSON(w, http.StatusOK, tasks)
}

// Logbook handles GET /views/logbook — returns tasks with status IN (1,2).
func (h *ViewHandler) Logbook(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.ViewLogbook(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	tasks := make([]*domain.Task, len(rows))
	for i, row := range rows {
		tasks[i] = viewTaskFromRow(row)
	}
	RespondJSON(w, http.StatusOK, tasks)
}
