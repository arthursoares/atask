package api

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/service"
)

// AuthHandler handles authentication and account HTTP routes. Identity
// operations (registration, login, refresh, provider discovery, profile
// CRUD) are delegated to the AuthProvider (PocketBase-backed). API-key
// management still goes through AuthService against the local api_keys
// table — Task 12 leaves that path untouched.
type AuthHandler struct {
	authProvider auth.AuthProvider
	authSvc      *service.AuthService
	cfg          *config.Config
}

// NewAuthHandler constructs an AuthHandler.
func NewAuthHandler(authProvider auth.AuthProvider, authSvc *service.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{authProvider: authProvider, authSvc: authSvc, cfg: cfg}
}

// RegisterRoutes registers all auth routes on the mux. Used by tests that
// drive a bare http.ServeMux directly (see decode_integration_test.go);
// production routing goes through RegisterRoutes in routes.go, which mounts
// the same handlers on PocketBase's router with per-route auth (register,
// login, refresh, and providers are public; everything else requires auth).
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("GET /auth/providers", h.Providers)
	mux.HandleFunc("GET /auth/me", h.GetMe)
	mux.HandleFunc("PUT /auth/me", h.UpdateMe)
	mux.HandleFunc("GET /auth/api-keys", h.ListAPIKeys)
	mux.HandleFunc("POST /auth/api-keys", h.CreateAPIKey)
	mux.HandleFunc("PUT /auth/api-keys/{id}", h.UpdateAPIKey)
	mux.HandleFunc("DELETE /auth/api-keys/{id}", h.DeleteAPIKey)
}

// userJSON is the wire representation shared by Register/Login/GetMe/UpdateMe.
func userJSON(u *auth.User) map[string]any {
	return map[string]any{
		"id":    u.ID,
		"email": u.Email,
		"name":  u.Name,
		"role":  u.Role,
	}
}

// extractBearerToken returns the token from an "Authorization: Bearer <token>"
// header, or "" if the header is missing or uses a different scheme.
func extractBearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimPrefix(header, prefix)
}

// Register handles POST /auth/register — creates a new user account via the
// AuthProvider. Registration is open in Phase 1: every account is created
// with role "user". Task 17 gates this behind config.RegistrationOpen /
// invites; until then this endpoint is intentionally permissive.
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

	user, err := h.authProvider.CreateUser(body.Email, body.Password, body.Name, "user")
	if err != nil {
		slog.Error("register: create user failed", "err", err)
		RespondError(w, http.StatusUnprocessableEntity, "could not create account")
		return
	}

	RespondJSON(w, http.StatusCreated, userJSON(user))
}

// Login handles POST /auth/login — returns a PocketBase auth token and the
// authenticated user's profile on success. AuthWithPassword returns the
// identical error for "unknown email" and "wrong password" (see
// internal/auth/pocketbase.go) so this handler cannot leak which one failed.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	token, user, err := h.authProvider.AuthWithPassword(body.Email, body.Password)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  userJSON(user),
	})
}

// Refresh handles POST /auth/refresh — exchanges a valid Bearer token for a
// newly minted one (PocketBase token rotation).
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		RespondError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	newToken, err := h.authProvider.RefreshToken(token)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, "refresh failed")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"token": newToken})
}

// Providers handles GET /auth/providers — reports which login providers are
// enabled (email is always on; OAuth providers depend on configured client
// IDs). This is backed by config, not the AuthProvider: PBAdapter's
// EnabledProviders is a vestigial interface member with no config access
// (see internal/auth/pocketbase.go) and is never called here.
func (h *AuthHandler) Providers(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, h.cfg.EnabledProviders())
}

// GetMe handles GET /auth/me — returns the authenticated user's profile.
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	user, err := h.authProvider.FindUserByID(userID)
	if err != nil {
		RespondError(w, http.StatusNotFound, "user not found")
		return
	}

	RespondJSON(w, http.StatusOK, userJSON(user))
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

	if err := h.authProvider.UpdateUser(userID, map[string]any{"name": body.Name}); err != nil {
		slog.Error("update me: update user failed", "err", err)
		RespondError(w, http.StatusUnprocessableEntity, "could not update profile")
		return
	}

	user, err := h.authProvider.FindUserByID(userID)
	if err != nil {
		slog.Error("update me: reload user failed", "err", err)
		RespondError(w, http.StatusInternalServerError, "could not update profile")
		return
	}

	RespondJSON(w, http.StatusOK, userJSON(user))
}

// ListAPIKeys handles GET /auth/api-keys — lists API keys for the current user.
func (h *AuthHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	if userID == "" {
		RespondError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	keys, err := h.authSvc.ListAPIKeys(r.Context(), userID)
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

	plainKey, apiKey, err := h.authSvc.CreateAPIKey(r.Context(), userID, body.Name)
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

	if err := h.authSvc.UpdateAPIKeyName(r.Context(), id, userID, body.Name); err != nil {
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
	if err := h.authSvc.DeleteAPIKey(r.Context(), id, userID); err != nil {
		RespondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"id": id})
}
