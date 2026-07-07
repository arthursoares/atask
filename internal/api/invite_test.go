package api_test

// Task 17: invite flow tests. These run against the same real-PocketBase-router
// + real-TCP-connection harness as pb_router_bridge_test.go / auth_test.go
// (startRealPBServerWithConfig), which wires a real auth.PBAdapter over
// tests.NewTestApp() as RoutesDeps.AuthProvider and a real *store.DB (in-memory
// SQLite) as RoutesDeps.DB — the invites table lives there.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
)

// inviteJSON mirrors CreateInvite's response shape (invite.go).
type inviteJSON struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	URL       string    `json:"url"`
}

// seedAdmin creates an admin-role user directly via the AuthProvider
// (bypassing POST /auth/register, which never assigns "admin" on any path —
// open registration hardcodes "user" and the invite path uses the invite's
// own role) and logs in over HTTP to obtain a usable Bearer token.
func seedAdmin(t *testing.T, srv *httptest.Server, ap auth.AuthProvider, email, password, name string) string {
	t.Helper()
	if _, err := ap.CreateUser(email, password, name, "admin"); err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	login := loginTestUser(t, srv, email, password)
	if login.Status != http.StatusOK {
		t.Fatalf("admin login failed: %d: %s", login.Status, login.RawBody)
	}
	return login.Token
}

// createInvite POSTs /auth/invites as the given (already-authenticated)
// bearer token and returns the raw response so callers can assert on status
// codes for both the happy path and the admin-gating failure paths.
func createInvite(t *testing.T, srv *httptest.Server, token, email, role string) (*http.Response, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "role": role})
	if err != nil {
		t.Fatalf("marshal invite body: %v", err)
	}
	req := bearerRequest(t, http.MethodPost, srv.URL+"/auth/invites", token, body)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST /auth/invites: %v", err)
	}
	defer resp.Body.Close()
	return resp, readBody(t, resp)
}

// claimInvite POSTs the public /auth/invites/claim endpoint.
func claimInvite(t *testing.T, srv *httptest.Server, token, password, name string) (*http.Response, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"token": token, "password": password, "name": name})
	if err != nil {
		t.Fatalf("marshal claim body: %v", err)
	}
	resp, err := http.Post(srv.URL+"/auth/invites/claim", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/invites/claim: %v", err)
	}
	defer resp.Body.Close()
	return resp, readBody(t, resp)
}

// TestInvite_CreateInvite_AdminOnly asserts POST /auth/invites is gated by
// requireAdminAPI (middleware.go): unauthenticated → 401, authenticated
// non-admin → 403, authenticated admin → 201 with an invite URL suffixed by
// the generated token.
func TestInvite_CreateInvite_AdminOnly(t *testing.T) {
	srv, app, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true, BaseURL: "http://localhost:8080"})
	ap := auth.NewPBAdapterFromApp(app)

	// Unauthenticated.
	unauthResp, unauthBody := createInviteNoAuth(t, srv, "nobody@example.com", "user")
	if unauthResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated CreateInvite, got %d: %s", unauthResp.StatusCode, unauthBody)
	}

	// Authenticated, non-admin.
	regularUser := registerTestUser(t, srv, "regular@example.com", "regularpass1", "Regular")
	regularLogin := loginTestUser(t, srv, regularUser.Email, "regularpass1")
	if regularLogin.Status != http.StatusOK {
		t.Fatalf("regular user login failed: %d: %s", regularLogin.Status, regularLogin.RawBody)
	}
	forbiddenResp, forbiddenBody := createInvite(t, srv, regularLogin.Token, "invitee@example.com", "user")
	if forbiddenResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin CreateInvite, got %d: %s", forbiddenResp.StatusCode, forbiddenBody)
	}

	// Authenticated admin: happy path.
	adminToken := seedAdmin(t, srv, ap, "admin@example.com", "adminpass1", "Admin")
	okResp, okBody := createInvite(t, srv, adminToken, "invitee@example.com", "user")
	if okResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for admin CreateInvite, got %d: %s", okResp.StatusCode, okBody)
	}
	var inv inviteJSON
	if err := json.Unmarshal([]byte(okBody), &inv); err != nil {
		t.Fatalf("decode invite response %q: %v", okBody, err)
	}
	if inv.Email != "invitee@example.com" || inv.Role != "user" {
		t.Errorf("unexpected invite fields: %+v", inv)
	}
	if inv.Token == "" {
		t.Fatal("expected a non-empty invite token")
	}
	wantSuffix := "/invite/" + inv.Token
	if len(inv.URL) < len(wantSuffix) || inv.URL[len(inv.URL)-len(wantSuffix):] != wantSuffix {
		t.Errorf("expected invite URL to end with %q, got %q", wantSuffix, inv.URL)
	}
	if inv.ExpiresAt.Before(time.Now().Add(6*24*time.Hour)) || inv.ExpiresAt.After(time.Now().Add(8*24*time.Hour)) {
		t.Errorf("expected expiresAt ~7 days out, got %v", inv.ExpiresAt)
	}
}

// createInviteNoAuth POSTs /auth/invites with no Authorization header.
func createInviteNoAuth(t *testing.T, srv *httptest.Server, email, role string) (*http.Response, string) {
	t.Helper()
	body, err := json.Marshal(map[string]string{"email": email, "role": role})
	if err != nil {
		t.Fatalf("marshal invite body: %v", err)
	}
	resp, err := http.Post(srv.URL+"/auth/invites", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/invites (no auth): %v", err)
	}
	defer resp.Body.Close()
	return resp, readBody(t, resp)
}

// TestInvite_CreateInvite_InvalidRole asserts an admin cannot mint an invite
// for any role other than "user"/"admin".
func TestInvite_CreateInvite_InvalidRole(t *testing.T) {
	srv, app, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})
	ap := auth.NewPBAdapterFromApp(app)
	adminToken := seedAdmin(t, srv, ap, "admin2@example.com", "adminpass2", "Admin2")

	resp, body := createInvite(t, srv, adminToken, "someone@example.com", "superadmin")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an invalid role, got %d: %s", resp.StatusCode, body)
	}
}

// TestInvite_ClaimInvite_HappyPathAndSingleUse covers the full claim flow:
// a valid token creates a usable account (login afterward succeeds with the
// invite's own email/role, not anything the claim request could smuggle in),
// and a second claim of the SAME token — whether framed as "already claimed"
// or "reused" — is rejected identically, since both are the same
// single-use-enforcement code path (AuthService.ClaimInvite's
// compare-and-swap UPDATE).
func TestInvite_ClaimInvite_HappyPathAndSingleUse(t *testing.T) {
	srv, app, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})
	ap := auth.NewPBAdapterFromApp(app)
	adminToken := seedAdmin(t, srv, ap, "admin3@example.com", "adminpass3", "Admin3")

	inviteResp, inviteBody := createInvite(t, srv, adminToken, "claimant@example.com", "user")
	if inviteResp.StatusCode != http.StatusCreated {
		t.Fatalf("create invite failed: %d: %s", inviteResp.StatusCode, inviteBody)
	}
	var inv inviteJSON
	if err := json.Unmarshal([]byte(inviteBody), &inv); err != nil {
		t.Fatalf("decode invite response %q: %v", inviteBody, err)
	}

	claimResp, claimBody := claimInvite(t, srv, inv.Token, "claimantpass1", "Claimant")
	if claimResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 claiming a valid invite, got %d: %s", claimResp.StatusCode, claimBody)
	}
	var claimed authUserJSON
	if err := json.Unmarshal([]byte(claimBody), &claimed); err != nil {
		t.Fatalf("decode claim response %q: %v", claimBody, err)
	}
	if claimed.Email != "claimant@example.com" || claimed.Role != "user" || claimed.Name != "Claimant" {
		t.Errorf("unexpected claimed user: %+v", claimed)
	}

	// The claimed account must actually be usable.
	login := loginTestUser(t, srv, "claimant@example.com", "claimantpass1")
	if login.Status != http.StatusOK {
		t.Fatalf("expected the claimed account to log in, got %d: %s", login.Status, login.RawBody)
	}

	// Reusing / re-claiming the same token must fail — single-use.
	reuseResp, reuseBody := claimInvite(t, srv, inv.Token, "anotherpass1", "Someone Else")
	if reuseResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 reusing an already-claimed invite token, got %d: %s", reuseResp.StatusCode, reuseBody)
	}
}

// TestInvite_ClaimInvite_Expired backdates a freshly created invite's
// expires_at directly in the domain DB (there is no public way to mint an
// already-expired invite through the API) and asserts the claim is rejected.
// This exercises AuthService.ValidateInviteToken's Go-side expiry recheck —
// the SQL predicate alone (`expires_at > datetime('now')`) is documented in
// invites.sql as unreliable under modernc.org/sqlite's time encoding.
func TestInvite_ClaimInvite_Expired(t *testing.T) {
	srv, app, db := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})
	ap := auth.NewPBAdapterFromApp(app)
	adminToken := seedAdmin(t, srv, ap, "admin4@example.com", "adminpass4", "Admin4")

	inviteResp, inviteBody := createInvite(t, srv, adminToken, "late@example.com", "user")
	if inviteResp.StatusCode != http.StatusCreated {
		t.Fatalf("create invite failed: %d: %s", inviteResp.StatusCode, inviteBody)
	}
	var inv inviteJSON
	if err := json.Unmarshal([]byte(inviteBody), &inv); err != nil {
		t.Fatalf("decode invite response %q: %v", inviteBody, err)
	}

	if _, err := db.DB.Exec(`UPDATE invites SET expires_at = ? WHERE token = ?`, time.Now().Add(-1*time.Hour), inv.Token); err != nil {
		t.Fatalf("backdate invite expiry: %v", err)
	}

	resp, body := claimInvite(t, srv, inv.Token, "latepass1", "Late")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 claiming an expired invite, got %d: %s", resp.StatusCode, body)
	}
}

// TestRegister_Gated_RequiresInvite covers the Register-side of the gate
// (auth.go's Register, config.RegistrationOpen == false): no invite token →
// 403, and a valid invite token succeeds by delegating to the same
// claimInvite helper ClaimInvite uses — the created account's email/role come
// from the invite, not from the (here, deliberately omitted) request body
// Email field.
func TestRegister_Gated_RequiresInvite(t *testing.T) {
	srv, app, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: false})
	ap := auth.NewPBAdapterFromApp(app)
	adminToken := seedAdmin(t, srv, ap, "admin5@example.com", "adminpass5", "Admin5")

	// No invite token: closed registration must reject.
	noInviteBody, err := json.Marshal(map[string]string{
		"email": "walkin@example.com", "password": "walkinpass1", "name": "Walkin",
	})
	if err != nil {
		t.Fatalf("marshal register body: %v", err)
	}
	noInviteResp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(noInviteBody))
	if err != nil {
		t.Fatalf("POST /auth/register (no invite): %v", err)
	}
	defer noInviteResp.Body.Close()
	noInviteRaw := readBody(t, noInviteResp)
	if noInviteResp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 registering without an invite while closed, got %d: %s", noInviteResp.StatusCode, noInviteRaw)
	}

	// With a valid invite: registration succeeds via the gated path.
	inviteResp, inviteBody := createInvite(t, srv, adminToken, "invited@example.com", "user")
	if inviteResp.StatusCode != http.StatusCreated {
		t.Fatalf("create invite failed: %d: %s", inviteResp.StatusCode, inviteBody)
	}
	var inv inviteJSON
	if err := json.Unmarshal([]byte(inviteBody), &inv); err != nil {
		t.Fatalf("decode invite response %q: %v", inviteBody, err)
	}

	gatedBody, err := json.Marshal(map[string]string{
		"password":    "invitedpass1",
		"name":        "Invited",
		"inviteToken": inv.Token,
	})
	if err != nil {
		t.Fatalf("marshal gated register body: %v", err)
	}
	gatedResp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(gatedBody))
	if err != nil {
		t.Fatalf("POST /auth/register (with invite): %v", err)
	}
	defer gatedResp.Body.Close()
	gatedRaw := readBody(t, gatedResp)
	if gatedResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 registering with a valid invite while closed, got %d: %s", gatedResp.StatusCode, gatedRaw)
	}
	var created authUserJSON
	if err := json.Unmarshal([]byte(gatedRaw), &created); err != nil {
		t.Fatalf("decode gated register response %q: %v", gatedRaw, err)
	}
	if created.Email != "invited@example.com" || created.Name != "Invited" || created.Role != "user" {
		t.Errorf("unexpected gated-register user: %+v (email/role must come from the invite)", created)
	}

	login := loginTestUser(t, srv, "invited@example.com", "invitedpass1")
	if login.Status != http.StatusOK {
		t.Fatalf("expected the gated-registered account to log in, got %d: %s", login.Status, login.RawBody)
	}
}

// TestRegister_Open_NoInviteNeeded is a light, dedicated regression check
// that config.RegistrationOpen == true never requires an invite token — the
// broader open-registration path is already covered end-to-end by
// TestAuth_RegisterLoginAndProtectedRoute (auth_test.go), which uses the same
// RegistrationOpen: true default (startRealPBServerWithApp).
func TestRegister_Open_NoInviteNeeded(t *testing.T) {
	srv, _, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})

	body, err := json.Marshal(map[string]string{
		"email": "open@example.com", "password": "openpass1", "name": "Open",
	})
	if err != nil {
		t.Fatalf("marshal register body: %v", err)
	}
	resp, err := http.Post(srv.URL+"/auth/register", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer resp.Body.Close()
	raw := readBody(t, resp)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for open registration with no invite token, got %d: %s", resp.StatusCode, raw)
	}
}
