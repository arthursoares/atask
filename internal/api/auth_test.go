package api_test

// Task 12: real AuthProvider-backed tests for the rewritten auth handlers
// (Login/Refresh/Providers/Register/GetMe/UpdateMe). These run against the
// same real-PocketBase-router + real-TCP-connection harness as
// pb_router_bridge_test.go (startRealPBServer, readBody — same package, same
// file), which already wires a real auth.PBAdapter (over
// tests.NewTestApp()) as RoutesDeps.AuthProvider.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// authUserJSON mirrors the {"id","email","name","role"} shape shared by
// Register/Login/GetMe/UpdateMe (see userJSON in auth.go).
type authUserJSON struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// registerTestUser POSTs /auth/register and fails the test unless it
// succeeds with 201, returning the decoded user.
func registerTestUser(t *testing.T, srv *httptest.Server, email, password, name string) authUserJSON {
	t.Helper()

	reqBody, err := json.Marshal(map[string]string{"email": email, "password": password, "name": name})
	if err != nil {
		t.Fatalf("marshal register body: %v", err)
	}

	resp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer resp.Body.Close()

	raw := readBody(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register %s: expected 201, got %d: %s", email, resp.StatusCode, raw)
	}

	var u authUserJSON
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		t.Fatalf("decode register response %q: %v", raw, err)
	}
	return u
}

// loginResult carries a /auth/login response for both the success and
// failure paths (a failed login has an empty Token but a populated Status
// and RawBody, so failure-path tests can assert on those).
type loginResult struct {
	Status  int
	RawBody string
	Token   string
	User    authUserJSON
}

// loginTestUser POSTs /auth/login and returns the raw result without
// failing the test — callers assert on Status/RawBody/Token themselves,
// since some tests (e.g. invalid-credential checks) expect a 401.
func loginTestUser(t *testing.T, srv *httptest.Server, email, password string) loginResult {
	t.Helper()

	reqBody, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		t.Fatalf("marshal login body: %v", err)
	}

	resp, err := http.Post(srv.URL+"/auth/login", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /auth/login: %v", err)
	}
	defer resp.Body.Close()

	raw := readBody(t, resp)
	result := loginResult{Status: resp.StatusCode, RawBody: raw}
	if resp.StatusCode != http.StatusOK {
		return result
	}

	var out struct {
		Token string       `json:"token"`
		User  authUserJSON `json:"user"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode login response %q: %v", raw, err)
	}
	result.Token = out.Token
	result.User = out.User
	return result
}

// bearerRequest builds an http.Request carrying "Authorization: Bearer <token>".
func bearerRequest(t *testing.T, method, url, token string, body []byte) *http.Request {
	t.Helper()

	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("build %s %s request: %v", method, url, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// TestAuth_RegisterLoginAndProtectedRoute covers the core happy path: an
// open POST /auth/register creates a usable account, POST /auth/login
// against it returns a token + user object, and that token authenticates a
// protected domain route (GET /tasks) via requireAuth's Bearer path.
func TestAuth_RegisterLoginAndProtectedRoute(t *testing.T) {
	srv := startRealPBServer(t)

	created := registerTestUser(t, srv, "alice@example.com", "hunter22pw", "Alice")
	if created.Email != "alice@example.com" || created.Name != "Alice" || created.Role != "user" {
		t.Fatalf("unexpected register response: %+v", created)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty user id from register")
	}

	login := loginTestUser(t, srv, "alice@example.com", "hunter22pw")
	if login.Status != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d: %s", login.Status, login.RawBody)
	}
	if login.Token == "" {
		t.Fatal("expected a non-empty token from login")
	}
	if login.User.Email != "alice@example.com" || login.User.ID != created.ID {
		t.Errorf("unexpected login user object: %+v", login.User)
	}

	req := bearerRequest(t, http.MethodGet, srv.URL+"/tasks", login.Token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from protected /tasks with a valid login token, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

// TestAuth_LoginFailure_NoEnumerationOracle asserts Login returns 401 for
// both an unknown email and a known email with the wrong password, with an
// identical response body — the HTTP-layer counterpart to
// internal/auth/pocketbase_test.go's TestAuthWithPassword_NoEnumerationOracle.
func TestAuth_LoginFailure_NoEnumerationOracle(t *testing.T) {
	srv := startRealPBServer(t)
	registerTestUser(t, srv, "bob@example.com", "correcthorse1", "Bob")

	unknown := loginTestUser(t, srv, "nope@example.com", "whatever12")
	wrongPass := loginTestUser(t, srv, "bob@example.com", "wrongpassword9")

	if unknown.Status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown email, got %d: %s", unknown.Status, unknown.RawBody)
	}
	if wrongPass.Status != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d: %s", wrongPass.Status, wrongPass.RawBody)
	}
	if unknown.RawBody != wrongPass.RawBody {
		t.Errorf("enumeration oracle: unknown-email body %q != wrong-password body %q", unknown.RawBody, wrongPass.RawBody)
	}
}

// TestAuth_RefreshRotatesToken asserts POST /auth/refresh mints a new token
// distinct from the one presented, and that the new token itself
// authenticates a protected route.
func TestAuth_RefreshRotatesToken(t *testing.T) {
	srv := startRealPBServer(t)
	registerTestUser(t, srv, "carol@example.com", "rotatetoken1", "Carol")
	login := loginTestUser(t, srv, "carol@example.com", "rotatetoken1")
	if login.Status != http.StatusOK {
		t.Fatalf("login failed: %d: %s", login.Status, login.RawBody)
	}

	// PocketBase's auth-token claims (id, collectionId, type, refreshable) are
	// identical between two calls for the same record, and "exp" is
	// time.Now().Add(duration).Unix() — second, not sub-second, granularity
	// (tools/security/jwt.go). Minting login's token and refresh's token
	// within the same wall-clock second therefore produces byte-identical
	// JWTs, not merely "likely equal" ones. Sleep past the second boundary so
	// the two tokens are guaranteed to differ, rather than asserting
	// something PocketBase doesn't actually promise at sub-second timescales.
	time.Sleep(1100 * time.Millisecond)

	req := bearerRequest(t, http.MethodPost, srv.URL+"/auth/refresh", login.Token, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /auth/refresh: %v", err)
	}
	defer resp.Body.Close()

	raw := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from refresh, got %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode refresh response %q: %v", raw, err)
	}
	if out.Token == "" {
		t.Fatal("expected a non-empty refreshed token")
	}
	if out.Token == login.Token {
		t.Error("expected refresh to mint a new token, got the same one back")
	}

	// The rotated token must itself authenticate a protected route.
	req2 := bearerRequest(t, http.MethodGet, srv.URL+"/tasks", out.Token, nil)
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("GET /tasks with refreshed token: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 using the refreshed token, got %d: %s", resp2.StatusCode, readBody(t, resp2))
	}
}

// TestAuth_Refresh_MissingOrInvalidToken asserts /auth/refresh is a public
// route (reachable without a pre-existing session) but still rejects a
// missing or garbage Bearer token with 401.
func TestAuth_Refresh_MissingOrInvalidToken(t *testing.T) {
	srv := startRealPBServer(t)

	noAuthReq, err := http.NewRequest(http.MethodPost, srv.URL+"/auth/refresh", bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(noAuthReq)
	if err != nil {
		t.Fatalf("POST /auth/refresh (no auth header): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no Authorization header, got %d: %s", resp.StatusCode, readBody(t, resp))
	}

	badReq := bearerRequest(t, http.MethodPost, srv.URL+"/auth/refresh", "not-a-real-token", nil)
	resp2, err := http.DefaultClient.Do(badReq)
	if err != nil {
		t.Fatalf("POST /auth/refresh (garbage token): %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a garbage token, got %d: %s", resp2.StatusCode, readBody(t, resp2))
	}
}

// TestAuth_Refresh_RejectsSuperuserToken asserts POST /auth/refresh — a
// public route — rejects a _superusers (admin) auth token with 401, mirroring
// ValidateToken's users-collection scoping guard
// (internal/auth/pocketbase.go's requireUsersCollectionRecord). Before the
// fix, RefreshToken only checked IsAuth() on the resolved record and would
// happily rotate a _superusers token into a fresh superuser token for any
// caller who obtained one and presented it to this public endpoint.
func TestAuth_Refresh_RejectsSuperuserToken(t *testing.T) {
	srv, app := startRealPBServerWithApp(t)

	// tests.NewTestApp seeds a test@example.com record in both the `users`
	// and `_superusers` collections (see pocketbase_test.go's newTestAdapter
	// doc comment) — mint a token for the superuser one.
	su, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "test@example.com")
	if err != nil {
		t.Fatalf("find superuser: %v", err)
	}
	suToken, err := su.NewAuthToken()
	if err != nil {
		t.Fatalf("superuser NewAuthToken: %v", err)
	}

	req := bearerRequest(t, http.MethodPost, srv.URL+"/auth/refresh", suToken, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /auth/refresh (superuser token): %v", err)
	}
	defer resp.Body.Close()
	raw := readBody(t, resp)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 rejecting a _superusers token, got %d: %s", resp.StatusCode, raw)
	}
}

// TestAuth_Register_RejectsRoleInjection is a regression test pinning what
// actually happens when a register request tries to smuggle in a "role"
// field: the Register body struct has no Role field and DecodeJSON sets
// DisallowUnknownFields (internal/api/response.go), so an unrecognized
// "role" key must be rejected with 400 rather than silently accepted (which
// would be safe only by accident, since Register always hardcodes "user"
// regardless of what's in the body).
func TestAuth_Register_RejectsRoleInjection(t *testing.T) {
	srv := startRealPBServer(t)

	reqBody, err := json.Marshal(map[string]string{
		"email":    "eve@example.com",
		"password": "roleinject1",
		"name":     "Eve",
		"role":     "admin",
	})
	if err != nil {
		t.Fatalf("marshal register body: %v", err)
	}

	resp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer resp.Body.Close()
	raw := readBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 rejecting the unknown \"role\" field, got %d: %s", resp.StatusCode, raw)
	}
}

// TestAuth_ProvidersShape asserts GET /auth/providers is public and reports
// email always enabled with the OAuth providers disabled when no client IDs
// are configured (startRealPBServer wires RoutesDeps.Config with no OAuth
// client IDs set — see startRealPBServerWithApp in pb_router_bridge_test.go).
func TestAuth_ProvidersShape(t *testing.T) {
	srv := startRealPBServer(t)

	resp, err := http.Get(srv.URL + "/auth/providers")
	if err != nil {
		t.Fatalf("GET /auth/providers: %v", err)
	}
	defer resp.Body.Close()

	raw := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, raw)
	}

	var providers map[string]bool
	if err := json.Unmarshal([]byte(raw), &providers); err != nil {
		t.Fatalf("decode providers response %q: %v", raw, err)
	}
	if !providers["email"] {
		t.Error("expected the email provider to be enabled")
	}
	if providers["google"] || providers["github"] {
		t.Errorf("expected oauth providers disabled with no client IDs configured, got: %+v", providers)
	}
}

// TestAuth_MeEndpoints covers GET /auth/me and PUT /auth/me end to end
// through a real Bearer token.
func TestAuth_MeEndpoints(t *testing.T) {
	srv := startRealPBServer(t)
	registerTestUser(t, srv, "dave@example.com", "meendpoint1", "Dave")
	login := loginTestUser(t, srv, "dave@example.com", "meendpoint1")
	if login.Status != http.StatusOK {
		t.Fatalf("login failed: %d: %s", login.Status, login.RawBody)
	}

	getReq := bearerRequest(t, http.MethodGet, srv.URL+"/auth/me", login.Token, nil)
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		t.Fatalf("GET /auth/me: %v", err)
	}
	defer getResp.Body.Close()
	getRaw := readBody(t, getResp)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from GET /auth/me, got %d: %s", getResp.StatusCode, getRaw)
	}

	var me authUserJSON
	if err := json.Unmarshal([]byte(getRaw), &me); err != nil {
		t.Fatalf("decode /auth/me response %q: %v", getRaw, err)
	}
	if me.Email != "dave@example.com" || me.Name != "Dave" {
		t.Errorf("unexpected GET /auth/me body: %+v", me)
	}

	updateBody, err := json.Marshal(map[string]string{"name": "David"})
	if err != nil {
		t.Fatalf("marshal update body: %v", err)
	}
	putReq := bearerRequest(t, http.MethodPut, srv.URL+"/auth/me", login.Token, updateBody)
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("PUT /auth/me: %v", err)
	}
	defer putResp.Body.Close()
	putRaw := readBody(t, putResp)
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from PUT /auth/me, got %d: %s", putResp.StatusCode, putRaw)
	}

	var updated authUserJSON
	if err := json.Unmarshal([]byte(putRaw), &updated); err != nil {
		t.Fatalf("decode PUT /auth/me response %q: %v", putRaw, err)
	}
	if updated.Name != "David" {
		t.Errorf("expected updated name %q, got %q", "David", updated.Name)
	}

	// A second GET must reflect the update (persisted, not just echoed).
	getReq2 := bearerRequest(t, http.MethodGet, srv.URL+"/auth/me", login.Token, nil)
	getResp2, err := http.DefaultClient.Do(getReq2)
	if err != nil {
		t.Fatalf("GET /auth/me (after update): %v", err)
	}
	defer getResp2.Body.Close()
	getRaw2 := readBody(t, getResp2)
	if getResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getResp2.StatusCode, getRaw2)
	}
	var me2 authUserJSON
	if err := json.Unmarshal([]byte(getRaw2), &me2); err != nil {
		t.Fatalf("decode second /auth/me response %q: %v", getRaw2, err)
	}
	if me2.Name != "David" {
		t.Errorf("expected persisted name %q on re-fetch, got %q", "David", me2.Name)
	}
}

// TestAuth_Me_RequiresAuth asserts /auth/me is not reachable without a valid
// Bearer token (it is registered as a protected route in routes.go).
func TestAuth_Me_RequiresAuth(t *testing.T) {
	srv := startRealPBServer(t)

	resp, err := http.Get(srv.URL + "/auth/me")
	if err != nil {
		t.Fatalf("GET /auth/me (no auth): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a Bearer token, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}
