package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
)

// csrfStore holds the set of currently-valid CSRF tokens for each admin
// session. It is process-memory only (a single mutex-guarded map): fine for the
// Phase 1 single-instance deployment, but see the horizontal-scaling note in
// the task report — a multi-instance admin panel would need a shared token
// store (Redis/DB) instead.
type csrfStore struct {
	mu     sync.RWMutex
	tokens map[string]map[string]struct{} // sessionID → set of valid CSRF tokens
}

// NewCSRFStore builds an empty in-memory CSRF token store.
func NewCSRFStore() *csrfStore { return &csrfStore{tokens: make(map[string]map[string]struct{})} }

// issue returns a fresh token AND adds it to the session's valid-token set.
// Multiple concurrent forms (e.g., two browser tabs) get distinct tokens that
// are all valid until consumed. This avoids the "second tab invalidates first"
// pitfall of single-token-per-session designs.
func (s *csrfStore) issue(sessionID string) string {
	buf := make([]byte, 32)
	rand.Read(buf)
	tok := hex.EncodeToString(buf)
	s.mu.Lock()
	if s.tokens[sessionID] == nil {
		s.tokens[sessionID] = make(map[string]struct{})
	}
	s.tokens[sessionID][tok] = struct{}{}
	s.mu.Unlock()
	return tok
}

// verify checks that the presented token is currently valid for this session
// AND consumes it (one-time use) so a captured token can't be replayed.
// Returns true on first valid presentation, false on any subsequent reuse.
func (s *csrfStore) verify(sessionID, presented string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	set, ok := s.tokens[sessionID]
	if !ok {
		return false
	}
	if _, valid := set[presented]; !valid {
		return false
	}
	delete(set, presented) // consume — single-use
	return true
}

// clear drops all valid tokens for a session (logout / session rotation).
func (s *csrfStore) clear(sessionID string) {
	s.mu.Lock()
	delete(s.tokens, sessionID)
	s.mu.Unlock()
}

// requireCSRF verifies (and consumes) a single-use CSRF token on every POST.
// Non-POST requests pass through untouched. A missing session cookie, an
// unparseable form, or a token that is not currently valid for the session all
// short-circuit with an error before the wrapped handler runs.
//
// Because tokens are single-use, this is applied INSIDE requireAdmin (after the
// session is known to exist) on the protected admin routes only.
func requireCSRF(store *csrfStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			sessionID, err := r.Cookie(adminSessionCookie)
			if err != nil {
				http.Error(w, "no session", http.StatusForbidden)
				return
			}
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad form", http.StatusBadRequest)
				return
			}
			if !store.verify(sessionID.Value, r.FormValue("csrf_token")) {
				http.Error(w, "your session expired, please retry", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
