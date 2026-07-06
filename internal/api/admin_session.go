package api

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// sessionStore maps an admin session ID (the opaque, unguessable cookie value)
// to the authenticated user ID. Like csrfStore it is process-memory only: fine
// for a single-instance Phase 1 deployment, but a horizontally-scaled admin
// panel would need a shared session backend (see the task report). Sessions are
// dropped on logout and on login-time rotation.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string // sessionID → userID
}

// NewSessionStore builds an empty in-memory session store.
func NewSessionStore() *sessionStore { return &sessionStore{sessions: make(map[string]string)} }

// Set binds a session ID to a user ID.
func (s *sessionStore) Set(sessionID, userID string) {
	s.mu.Lock()
	s.sessions[sessionID] = userID
	s.mu.Unlock()
}

// Get returns the user ID for a session ID, and whether it exists.
func (s *sessionStore) Get(sessionID string) (string, bool) {
	s.mu.RLock()
	userID, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	return userID, ok
}

// Delete removes a session (logout / rotation).
func (s *sessionStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// generateSessionID returns a 32-byte, base64url-encoded random session ID.
// The value is the cookie the browser presents; it must be unguessable so an
// attacker cannot forge or fixate a session.
func generateSessionID() string {
	buf := make([]byte, 32)
	rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}
