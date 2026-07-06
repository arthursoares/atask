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
	userID := UserIDFromContext(r.Context())
	var body struct {
		Title string `json:"title"`
		ID    string `json:"id,omitempty"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	tag, err := h.tags.Create(r.Context(), userID, body.Title, actorFromRequest(r), body.ID)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.TagCreated), tag)
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	tags, err := h.tags.List(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, tags)
}

func (h *TagHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")
	tag, err := h.tags.Get(r.Context(), userID, id)
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
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	if err := h.tags.Rename(r.Context(), userID, id, body.Title, actorFromRequest(r)); err != nil {
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
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")

	if err := h.tags.Delete(r.Context(), userID, id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "tag not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.TagDeleted), map[string]string{"id": id})
}
