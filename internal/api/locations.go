package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/atask/atask/internal/domain"
	"github.com/atask/atask/internal/service"
)

// LocationHandler holds the LocationService and handles location HTTP routes.
type LocationHandler struct {
	locations *service.LocationService
}

// NewLocationHandler constructs a LocationHandler.
func NewLocationHandler(locations *service.LocationService) *LocationHandler {
	return &LocationHandler{locations: locations}
}

// RegisterRoutes registers all location routes on the mux.
func (h *LocationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /locations", h.Create)
	mux.HandleFunc("GET /locations", h.List)
	mux.HandleFunc("GET /locations/{id}", h.Get)
	mux.HandleFunc("PUT /locations/{id}", h.Rename)
	mux.HandleFunc("DELETE /locations/{id}", h.Delete)
}

func (h *LocationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	var body struct {
		Title string `json:"title"`
		Name  string `json:"name"`
		ID    string `json:"id,omitempty"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}
	name := body.Name
	if name == "" {
		name = body.Title
	}

	loc, err := h.locations.Create(r.Context(), userID, name, actorFromRequest(r), body.ID)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusCreated, string(domain.LocationCreated), loc)
}

func (h *LocationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	locs, err := h.locations.List(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, locs)
}

func (h *LocationHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")
	loc, err := h.locations.Get(r.Context(), userID, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "location not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	RespondJSON(w, http.StatusOK, loc)
}

func (h *LocationHandler) Rename(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")
	var body struct {
		Title string `json:"title"`
		Name  string `json:"name"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}
	name := body.Name
	if name == "" {
		name = body.Title
	}

	if err := h.locations.Rename(r.Context(), userID, id, name, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "location not found")
			return
		}
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.LocationRenamed), map[string]string{"id": id})
}

func (h *LocationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	id := r.PathValue("id")

	if err := h.locations.Delete(r.Context(), userID, id, actorFromRequest(r)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			RespondError(w, http.StatusNotFound, "location not found")
			return
		}
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondEvent(w, http.StatusOK, string(domain.LocationDeleted), map[string]string{"id": id})
}
