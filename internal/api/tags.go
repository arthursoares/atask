package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// TagHandler holds the TagService and handles tag HTTP routes.
type TagHandler struct {
	tags *service.TagService
}

// NewTagHandler constructs a TagHandler.
func NewTagHandler(tags *service.TagService) *TagHandler {
	return &TagHandler{tags: tags}
}

// RegisterRoutes registers all tag routes on the mux.
func (h *TagHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /tags", h.Create)
	mux.HandleFunc("GET /tags", h.List)
	mux.HandleFunc("GET /tags/{id}", h.Get)
	mux.HandleFunc("PUT /tags/{id}", h.Rename)
	mux.HandleFunc("DELETE /tags/{id}", h.Delete)
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
		ID    string `json:"id,omitempty"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	tag, err := h.tags.Create(r.Context(), body.Title, actorFromRequest(r), body.ID)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.TagCreated), tag)
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.tags.List(r.Context())
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, tags)
}

func (h *TagHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := h.tags.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "tag not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, tag)
}

func (h *TagHandler) Rename(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := h.tags.Rename(r.Context(), id, body.Title, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "tag not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TagRenamed), map[string]string{"id": id})
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.tags.Delete(r.Context(), id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "tag not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TagDeleted), map[string]string{"id": id})
}
