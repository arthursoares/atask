package api

import (
	"net/http"

	"github.com/atask/atask/internal/service"
)

// AuthHandler holds the AuthService and handles authentication HTTP routes.
type AuthHandler struct {
	auth *service.AuthService
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// RegisterRoutes registers all auth routes on the mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("GET /auth/me", h.GetMe)
	mux.HandleFunc("PUT /auth/me", h.UpdateMe)
	mux.HandleFunc("GET /auth/api-keys", h.ListAPIKeys)
	mux.HandleFunc("POST /auth/api-keys", h.CreateAPIKey)
	mux.HandleFunc("PUT /auth/api-keys/{id}", h.UpdateAPIKey)
	mux.HandleFunc("DELETE /auth/api-keys/{id}", h.DeleteAPIKey)
}

// Register handles POST /auth/register — creates a new user account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	user, err := h.auth.CreateUser(r.Context(), body.Email, body.Password, body.Name)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondJSON(w, http.StatusCreated, user)
}

// Login handles POST /auth/login — returns a signed JWT on success.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	token, err := h.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"token": token})
}

// GetMe handles GET /auth/me — returns the authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "user not found")
		return
	}

	RespondJSON(w, http.StatusOK, user)
}

// UpdateMe handles PUT /auth/me — updates the authenticated user's name.
func (h *AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	if err := h.auth.UpdateUser(r.Context(), userID, body.Name); err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, user)
}

// ListAPIKeys handles GET /auth/api-keys — lists API keys for the current user.
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	keys, err := h.auth.ListAPIKeys(r.Context(), userID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, keys)
}

// CreateAPIKey handles POST /auth/api-keys — creates a new API key for the current user.
// The plaintext key is returned once and cannot be retrieved again.
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	plainKey, apiKey, err := h.auth.CreateAPIKey(r.Context(), userID, body.Name)
	if err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]any{
		"key":     plainKey,
		"api_key": apiKey,
	})
}

// UpdateAPIKey handles PUT /auth/api-keys/{id} — renames an API key.
func (h *AuthHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id := r.PathValue("id")
	var body struct {
		Name string `json:"name"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	if err := h.auth.UpdateAPIKeyName(r.Context(), id, userID, body.Name); err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"id": id})
}

// DeleteAPIKey handles DELETE /auth/api-keys/{id} — deletes an API key.
func (h *AuthHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	id := r.PathValue("id")
	if err := h.auth.DeleteAPIKey(r.Context(), id, userID); err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"id": id})
}
