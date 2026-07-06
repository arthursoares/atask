package api

import (
	"context"
	"net/http"
)

// WithTestUser returns middleware that injects userID into the request
// context under the same key the production Auth middleware uses, so
// handlers under test see an authenticated request without needing a real
// JWT/API-key flow.
//
// Exported (capitalized) so external test files in package api_test — where
// most handler test suites live — can wrap their test servers with it, e.g.:
//
//	mux := http.NewServeMux()
//	handler.RegisterRoutes(mux)
//	return api.WithTestUser("test-user-1")(mux)
//
// Tests exercising cross-user isolation (Task 7) construct their own
// per-request wrapper instead of using a single fixed test user.
func WithTestUser(userID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ctxUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
