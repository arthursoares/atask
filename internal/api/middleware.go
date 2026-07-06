package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atask/atask/internal/auth"
	"github.com/google/uuid"
)

// APIKeyValidator is the subset of AuthService that requireAuth depends on.
// ValidateAPIKey returns the owning user ID, the key ID (for actor attribution),
// and the key's scope. *service.AuthService satisfies this interface.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, key string) (userID, keyID, scope string, err error)
}

// responseWriter wraps http.ResponseWriter to capture the written status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying ResponseWriter if it supports http.Flusher
// (required for SSE streaming).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying ResponseWriter, allowing http.ResponseController
// to access connection-level features like SetWriteDeadline.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Logging wraps a handler and logs each request with method, path, status
// code, and duration using slog.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start).String(),
		)
	})
}

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	ctxUserID    contextKey = "userID"
	ctxKeyID     contextKey = "keyID"
)

// RequestID middleware adds a unique request ID to the request context and
// sets the X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext retrieves the request ID stored in ctx by the
// RequestID middleware.  It returns an empty string when no ID is present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// UserIDFromContext returns the authenticated user ID stored in ctx by the
// Auth middleware. Returns an empty string when not present.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxUserID).(string)
	return id
}

// KeyIDFromContext returns the API key ID stored in ctx by the Auth middleware.
// Returns an empty string when the request was not authenticated via API key.
func KeyIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyID).(string)
	return id
}

// requireAuth is middleware that validates Bearer tokens (PocketBase auth tokens,
// resolved via the AuthProvider) and ApiKey credentials (resolved via the local
// api_keys table). It expects an Authorization header of the form "Bearer {token}"
// or "ApiKey {key}".
//
// Security invariants (each covered by a test in middleware_test.go):
//   - Missing/unsupported scheme or invalid credentials → 401.
//   - An empty resolved userID → 401. An empty subject must never become an
//     implicit owner of the pre-migration ” data pool.
//   - After resolving a userID, the user record is loaded from the identity
//     backend. A missing record (orphaned API key, deleted user) → 401; a
//     record with Disabled=true → 403.
//
// On the ApiKey path the key ID is stored in the request context so that
// actorFromRequest can attribute mutations to the agent's key rather than the
// owning user.
func requireAuth(authProvider auth.AuthProvider, apiKeySvc APIKeyValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")

			ctx := r.Context()
			var userID, keyID, scope string
			var isAPIKeyAuth bool
			var err error

			switch {
			case strings.HasPrefix(header, "ApiKey "):
				key := strings.TrimPrefix(header, "ApiKey ")
				userID, keyID, scope, err = apiKeySvc.ValidateAPIKey(ctx, key)
				isAPIKeyAuth = true
			case strings.HasPrefix(header, "Bearer "):
				token := strings.TrimPrefix(header, "Bearer ")
				userID, err = authProvider.ValidateToken(token)
			default:
				RespondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			if err != nil {
				RespondError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}

			// Reject an empty resolved subject before it can masquerade as an owner.
			if userID == "" {
				RespondError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}

			// Load the user record from the identity backend. A missing record
			// (deleted user / orphaned API key) is a 401 — safer than 403 and
			// consistent with the empty-userID rejection above. A disabled but
			// existing account is a 403.
			user, ferr := authProvider.FindUserByID(userID)
			if ferr != nil {
				RespondError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			if user.Disabled {
				RespondError(w, http.StatusForbidden, "account disabled")
				return
			}

			// Scope enforcement applies only to the ApiKey auth path — Bearer
			// tokens are PocketBase-authenticated humans and are always
			// full-access. api_keys.scope is loaded by ValidateAPIKey
			// (migration 006 / Task 1.5); expiry is enforced further upstream
			// by the same call (GetAPIKeyByHash's SQL predicate excludes
			// expired rows, surfacing as the "invalid credentials" 401 above).
			if isAPIKeyAuth {
				switch scope {
				case "read":
					if r.Method != http.MethodGet && r.Method != http.MethodHead {
						RespondError(w, http.StatusForbidden, "api key has read-only scope")
						return
					}
				case "read_write":
					// Domain endpoints OK; no admin API endpoints exist yet.
				case "admin":
					// All endpoints OK.
				default:
					RespondError(w, http.StatusForbidden, "unknown scope")
					return
				}
			}

			ctx = context.WithValue(ctx, ctxUserID, userID)
			if keyID != "" {
				ctx = context.WithValue(ctx, ctxKeyID, keyID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireAdminAPI gates admin-only JSON API endpoints (as opposed to the
// Task 14 web admin UI, which uses session cookies and requireAdmin in
// admin.go). It must run AFTER requireAuth — which resolves the Bearer/ApiKey
// credential and populates ctxUserID — and re-loads the user record via
// FindUserByID to check Role == "admin", mirroring requireAdmin's own
// re-check of the role rather than trusting a stale claim.
//
// There is no prior admin-API pattern in this codebase to follow (Task 14's
// admin surface is entirely session/cookie-based); this is a new, minimal
// convention introduced for Task 17's POST /auth/invites and intended to be
// reused by any future admin-only JSON endpoint rather than re-derived.
//
// An unauthenticated caller (ctxUserID empty) is rejected with 401 — this
// only happens if requireAdminAPI is ever wired without requireAuth ahead of
// it, since requireAuth itself already 401s before ctxUserID would be unset.
// A missing user record is 401; an authenticated non-admin is 403.
func requireAdminAPI(authProvider auth.AuthProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := UserIDFromContext(r.Context())
			if userID == "" {
				RespondError(w, http.StatusUnauthorized, "not authenticated")
				return
			}
			user, err := authProvider.FindUserByID(userID)
			if err != nil {
				RespondError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			if user.Role != "admin" {
				RespondError(w, http.StatusForbidden, "admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
