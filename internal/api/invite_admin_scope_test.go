package api_test

// Codex P1 follow-up: requireAdminAPI (middleware.go) previously checked only
// the authenticated user's Role, entirely ignoring API-key scope. That let an
// admin user's read_write-scoped API key reach POST /auth/invites (registered
// under requireAdminAPI in routes.go) even though the *key* was never granted
// admin-API access — only a Bearer token (a human) or a genuinely
// admin-scoped key should be able to mint invites. These tests exercise the
// fix through the same real-PocketBase-router + real-TCP-connection harness
// as invite_test.go (startRealPBServerWithConfig), minting API keys with an
// arbitrary scope directly against the domain DB (AuthService.CreateAPIKey
// hardcodes scope "read_write" — see middleware_scope_test.go's identical
// mintKey helper, unusable here since it lives in package api, not api_test).

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
)

// mintAPIKeyWithScope inserts an api_keys row directly against db with an
// arbitrary scope for an arbitrary user, and returns the plaintext key that
// hashes to the stored key_hash — exactly what a client presents as
// "ApiKey <key>".
func mintAPIKeyWithScope(t *testing.T, db *store.DB, userID, scope string) string {
	t.Helper()
	q := sqlc.New(db.DB)

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
		Name:        sql.NullString{String: "admin-scope-test-key", Valid: true},
		KeyHash:     sql.NullString{String: keyHash, Valid: true},
		Permissions: "[]",
		Scope:       scope,
		CreatedAt:   sql.NullTime{Time: time.Now(), Valid: true},
	}
	if _, err := q.CreateAPIKey(context.Background(), params); err != nil {
		t.Fatalf("insert api key: %v", err)
	}
	return plainKey
}

// apiKeyRequest builds an http.Request carrying "Authorization: ApiKey <key>"
// (bearerRequest in auth_test.go only builds Bearer requests).
func apiKeyRequest(t *testing.T, method, url, key string, body []byte) *http.Request {
	t.Helper()

	var reader io.Reader = bytes.NewReader(nil)
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("build %s %s request: %v", method, url, err)
	}
	req.Header.Set("Authorization", "ApiKey "+key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// createInviteWithAPIKey POSTs /auth/invites using an "ApiKey <key>"
// Authorization header.
func createInviteWithAPIKey(t *testing.T, srv *httptest.Server, key, email, role string) (*http.Response, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "role": role})
	if err != nil {
		t.Fatalf("marshal invite body: %v", err)
	}
	req := apiKeyRequest(t, http.MethodPost, srv.URL+"/auth/invites", key, body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /auth/invites (api key): %v", err)
	}
	defer resp.Body.Close()
	return resp, readBody(t, resp)
}

// TestInvite_CreateInvite_ReadWriteAPIKeyForbidden is the core regression
// guard for the bug: an admin user's read_write-scoped API key must NOT be
// able to mint invites, even though the user's Role is "admin".
func TestInvite_CreateInvite_ReadWriteAPIKeyForbidden(t *testing.T) {
	srv, app, db := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true, BaseURL: "http://localhost:8080"})
	ap := auth.NewPBAdapterFromApp(app)

	admin, err := ap.CreateUser("admin-rw-key@example.com", "adminrwpass1", "AdminRW", "admin")
	if err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	key := mintAPIKeyWithScope(t, db, admin.ID, "read_write")

	resp, body := createInviteWithAPIKey(t, srv, key, "invitee-rw@example.com", "user")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for an admin's read_write-scoped API key, got %d: %s", resp.StatusCode, body)
	}
}

// TestInvite_CreateInvite_AdminScopedAPIKeySucceeds proves the fix isn't
// over-broad: an admin's admin-scoped API key must still be able to mint
// invites.
func TestInvite_CreateInvite_AdminScopedAPIKeySucceeds(t *testing.T) {
	srv, app, db := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true, BaseURL: "http://localhost:8080"})
	ap := auth.NewPBAdapterFromApp(app)

	admin, err := ap.CreateUser("admin-admin-key@example.com", "adminadminpass1", "AdminAdmin", "admin")
	if err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	key := mintAPIKeyWithScope(t, db, admin.ID, "admin")

	resp, body := createInviteWithAPIKey(t, srv, key, "invitee-admin@example.com", "user")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for an admin's admin-scoped API key, got %d: %s", resp.StatusCode, body)
	}
}

// TestInvite_CreateInvite_NonAdminAdminScopedAPIKeyForbidden proves the scope
// check is additive to (not a replacement for) the existing Role check: a
// non-admin user's admin-scoped API key must still be rejected.
func TestInvite_CreateInvite_NonAdminAdminScopedAPIKeyForbidden(t *testing.T) {
	srv, app, db := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})
	ap := auth.NewPBAdapterFromApp(app)

	regular, err := ap.CreateUser("regular-admin-key@example.com", "regularpass1", "RegularAdminKey", "user")
	if err != nil {
		t.Fatalf("seed regular user: %v", err)
	}
	key := mintAPIKeyWithScope(t, db, regular.ID, "admin")

	resp, body := createInviteWithAPIKey(t, srv, key, "invitee-x@example.com", "user")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-admin's admin-scoped API key, got %d: %s", resp.StatusCode, body)
	}
}

// TestInvite_ReadWriteAPIKey_StillWorksOnDomainEndpoints is the regression
// guard proving the fix is scoped to requireAdminAPI: a read_write API key
// minted through the normal production endpoint (POST /auth/api-keys) must
// still work on ordinary domain endpoints (GET /tasks), unaffected by
// requireAdminAPI's new scope check.
func TestInvite_ReadWriteAPIKey_StillWorksOnDomainEndpoints(t *testing.T) {
	srv, _, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})

	user := registerTestUser(t, srv, "domain-rw-key@example.com", "domainpass1", "DomainUser")
	login := loginTestUser(t, srv, user.Email, "domainpass1")
	if login.Status != http.StatusOK {
		t.Fatalf("login failed: %d: %s", login.Status, login.RawBody)
	}

	keyBody, err := json.Marshal(map[string]string{"name": "domain-test-key"})
	if err != nil {
		t.Fatalf("marshal api key body: %v", err)
	}
	keyResp, err := http.DefaultClient.Do(bearerRequest(t, http.MethodPost, srv.URL+"/auth/api-keys", login.Token, keyBody))
	if err != nil {
		t.Fatalf("POST /auth/api-keys: %v", err)
	}
	defer keyResp.Body.Close()
	keyRaw := readBody(t, keyResp)
	if keyResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating an api key, got %d: %s", keyResp.StatusCode, keyRaw)
	}
	var out struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(keyRaw), &out); err != nil {
		t.Fatalf("decode api key response %q: %v", keyRaw, err)
	}

	tasksResp, err := http.DefaultClient.Do(apiKeyRequest(t, http.MethodGet, srv.URL+"/tasks", out.Key, nil))
	if err != nil {
		t.Fatalf("GET /tasks (api key): %v", err)
	}
	defer tasksResp.Body.Close()
	tasksRaw := readBody(t, tasksResp)
	if tasksResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for a read_write key on GET /tasks, got %d: %s", tasksResp.StatusCode, tasksRaw)
	}
}
