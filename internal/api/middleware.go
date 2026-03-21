package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/atask/atask/internal/service"
	"github.com/google/uuid"
)

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

// Auth is middleware that validates Bearer JWT tokens and ApiKey credentials.
// It expects an Authorization header of the form "Bearer {token}" or
// "ApiKey {key}". Requests without valid credentials receive a 401 response.
func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				RespondError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			ctx := r.Context()

			switch {
			case strings.HasPrefix(header, "Bearer "):
				token := strings.TrimPrefix(header, "Bearer ")
				userID, err := authService.ValidateToken(token)
				if err != nil {
					RespondError(w, http.StatusUnauthorized, "invalid token")
					return
				}
				ctx = context.WithValue(ctx, ctxUserID, userID)

			case strings.HasPrefix(header, "ApiKey "):
				key := strings.TrimPrefix(header, "ApiKey ")
				userID, keyID, err := authService.ValidateAPIKey(ctx, key)
				if err != nil {
					RespondError(w, http.StatusUnauthorized, "invalid api key")
					return
				}
				ctx = context.WithValue(ctx, ctxUserID, userID)
				ctx = context.WithValue(ctx, ctxKeyID, keyID)

			default:
				RespondError(w, http.StatusUnauthorized, "unsupported authorization scheme")
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
