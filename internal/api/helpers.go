package api

import "net/http"

// actorFromRequest returns the actor ID for the current request.
// It prefers the API key ID (for agent attribution), then the user ID,
// and falls back to "system" when no auth context is present.
func actorFromRequest(r *http.Request) string {
	if keyID := KeyIDFromContext(r.Context()); keyID != "" {
		return keyID
	}
	if userID := UserIDFromContext(r.Context()); userID != "" {
		return userID
	}
	return "system"
}
