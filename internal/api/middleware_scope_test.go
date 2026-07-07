package api

// Task 16: API key scope enforcement. These tests exercise requireAuth's
// scope branch through a *real* service.AuthService.ValidateAPIKey backed by
// an in-memory SQLite DB (not the fakeAPIKeyValidator used elsewhere in this
// package), so the expiry predicate from Task 1.5 (GetAPIKeyByHash's SQL
// WHERE clause) is genuinely exercised rather than assumed.
//
// AuthService.CreateAPIKey hardcodes scope "read_write" (Task 11), so tests
// that need a non-default scope or an expiry insert the api_keys row
// directly via sqlc's generated CreateAPIKey query — the same query
// AuthService itself uses — rather than extending CreateAPIKey's signature
// for a need only tests have so far.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
)

// scopeTestHarness wires a real AuthService (and therefore a real
// ValidateAPIKey) against a fresh in-memory DB, plus a fakeAuthProvider that
// reports the given user as enabled.
type scopeTestHarness struct {
	queries *sqlc.Queries
	authSvc *service.AuthService
	authP   *fakeAuthProvider
}

func newScopeTestHarness(t *testing.T, userID string) *scopeTestHarness {
	t.Helper()

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("store.NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	return &scopeTestHarness{
		queries: sqlc.New(db.DB),
		authSvc: service.NewAuthService(db, "test-secret"),
		authP:   &fakeAuthProvider{users: map[string]*auth.User{userID: enabledUser(userID)}},
	}
}

// mintKey inserts an api_keys row directly (the same CreateAPIKey sqlc query
// AuthService.CreateAPIKey uses internally) with an arbitrary scope and
// optional expiry, and returns the plaintext key that hashes to the stored
// key_hash — exactly what a client presents as "ApiKey <key>".
func (h *scopeTestHarness) mintKey(t *testing.T, userID, scope string, expiresAt *time.Time) string {
	t.Helper()

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	plainKey := hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(plainKey))
	keyHash := hex.EncodeToString(sum[:])

	params := sqlc.CreateAPIKeyParams{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        sql.NullString{String: "scope-test-key", Valid: true},
		KeyHash:     sql.NullString{String: keyHash, Valid: true},
		Permissions: "[]",
		Scope:       scope,
		CreatedAt:   sql.NullTime{Time: time.Now(), Valid: true},
	}
	if expiresAt != nil {
		params.ExpiresAt = sql.NullTime{Time: *expiresAt, Valid: true}
	}

	if _, err := h.queries.CreateAPIKey(context.Background(), params); err != nil {
		t.Fatalf("insert api key: %v", err)
	}
	return plainKey
}

// methodEchoHandler is a stand-in "next" handler for scope tests: it returns
// 201 for POST (mimicking a Create endpoint) and 200 for everything else
// (mimicking List/Get), so tests can tell whether requireAuth let the request
// reach the domain handler at all, and which one it reached.
type methodEchoHandler struct{ served bool }

func (h *methodEchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.served = true
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// doAPIKeyRequest drives requireAuth with an "ApiKey <key>" Authorization header.
func doAPIKeyRequest(mw func(http.Handler) http.Handler, next http.Handler, method, path, apiKey string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "ApiKey "+apiKey)
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec
}

func TestAPIKeyScope_ReadOnlyRejectsPost(t *testing.T) {
	const userID = "user-scope-1"
	h := newScopeTestHarness(t, userID)
	key := h.mintKey(t, userID, "read", nil)
	next := &methodEchoHandler{}

	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), next, http.MethodPost, "/tasks", key)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if next.served {
		t.Error("handler must not be served for a read-scope POST")
	}
}

// TestAPIKeyScope_ReadOnlyAllowsGet proves the read-scope check isn't
// over-broad: a GET with a read-scope key must still reach the handler.
func TestAPIKeyScope_ReadOnlyAllowsGet(t *testing.T) {
	const userID = "user-scope-2"
	h := newScopeTestHarness(t, userID)
	key := h.mintKey(t, userID, "read", nil)
	next := &methodEchoHandler{}

	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), next, http.MethodGet, "/tasks", key)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !next.served {
		t.Error("handler should be served for a read-scope GET")
	}
}

func TestAPIKeyScope_ReadWriteAllowsPostAndGet(t *testing.T) {
	const userID = "user-scope-3"
	h := newScopeTestHarness(t, userID)
	key := h.mintKey(t, userID, "read_write", nil)

	postNext := &methodEchoHandler{}
	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), postNext, http.MethodPost, "/tasks", key)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for read_write POST, got %d: %s", rec.Code, rec.Body.String())
	}

	getNext := &methodEchoHandler{}
	rec = doAPIKeyRequest(requireAuth(h.authP, h.authSvc), getNext, http.MethodGet, "/tasks", key)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for read_write GET, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIKeyScope_AdminAllowsPost(t *testing.T) {
	const userID = "user-scope-4"
	h := newScopeTestHarness(t, userID)
	key := h.mintKey(t, userID, "admin", nil)
	next := &methodEchoHandler{}

	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), next, http.MethodPost, "/tasks", key)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for admin POST, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAPIKeyScope_UnknownScopeRejected(t *testing.T) {
	const userID = "user-scope-5"
	h := newScopeTestHarness(t, userID)
	key := h.mintKey(t, userID, "bogus", nil)
	next := &methodEchoHandler{}

	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), next, http.MethodGet, "/tasks", key)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for unknown scope, got %d: %s", rec.Code, rec.Body.String())
	}
	if next.served {
		t.Error("handler must not be served for an unknown scope")
	}
}

// TestAPIKeyScope_ExpiredRejected: expiry is enforced inside ValidateAPIKey
// via the SQL predicate from Task 1.5 (GetAPIKeyByHash excludes rows whose
// expires_at has passed), so an expired key looks identical to an unknown one
// — ValidateAPIKey returns an error and requireAuth maps that to 401, never
// reaching the scope switch at all.
func TestAPIKeyScope_ExpiredRejected(t *testing.T) {
	const userID = "user-scope-6"
	h := newScopeTestHarness(t, userID)
	past := time.Now().Add(-1 * time.Hour)
	key := h.mintKey(t, userID, "read_write", &past)
	next := &methodEchoHandler{}

	rec := doAPIKeyRequest(requireAuth(h.authP, h.authSvc), next, http.MethodGet, "/tasks", key)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for an expired key, got %d: %s", rec.Code, rec.Body.String())
	}
	if next.served {
		t.Error("handler must not be served for an expired key")
	}
}

// TestAPIKeyScope_BearerUnaffectedByScope proves scope enforcement is scoped
// to the ApiKey auth path: a Bearer-authenticated POST succeeds even when the
// (unused) APIKeyValidator would report scope "read", and ValidateAPIKey is
// never even called.
func TestAPIKeyScope_BearerUnaffectedByScope(t *testing.T) {
	ap := &fakeAuthProvider{tokenUserID: "user-bearer", users: map[string]*auth.User{"user-bearer": enabledUser("user-bearer")}}
	kv := &fakeAPIKeyValidator{scope: "read"} // would 403 a POST if scope logic leaked into the bearer path
	next := &methodEchoHandler{}

	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer sometoken")
	rec := httptest.NewRecorder()
	requireAuth(ap, kv)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 for bearer POST regardless of api key scope, got %d: %s", rec.Code, rec.Body.String())
	}
	if kv.called {
		t.Error("ValidateAPIKey should not be called on the bearer path")
	}
	if !next.served {
		t.Error("handler should be served for a bearer POST")
	}
}
