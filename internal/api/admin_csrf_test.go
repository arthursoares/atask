package api_test

// Task 14, Step 6: CSRF protection test suite for the web admin UI. These build
// on the setupAdminServer / loginAsAdmin / fetchCSRFToken / postAdminForm
// helpers in admin_test.go.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// newUserForm builds a create-user form body carrying the given CSRF token.
// Each call uses a distinct email so repeated successful creates don't collide.
func newUserForm(csrfToken, email string) url.Values {
	return url.Values{
		"csrf_token": {csrfToken},
		"email":      {email},
		"name":       {"Test"},
		"password":   {"newuserpw12"},
		"role":       {"user"},
	}
}

// TestAdminCSRF_RejectsPostWithoutToken: a POST to a protected mutation route
// carrying a valid session but no csrf_token must be rejected with 403.
func TestAdminCSRF_RejectsPostWithoutToken(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	form := url.Values{"email": {"foo@bar.com"}, "name": {"Foo"}, "role": {"user"}, "password": {"foopass1234"}}
	resp := postAdminForm(t, env, session, "/admin/users/new", form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a POST with no csrf_token, got %d", resp.StatusCode)
	}
}

// TestAdminCSRF_AcceptsPostWithValidToken: GET the form to obtain a token, then
// POST it back — expect the 303 redirect the success path returns.
func TestAdminCSRF_AcceptsPostWithValidToken(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	token := fetchCSRFToken(t, env, session, "/admin/users/new")
	resp := postAdminForm(t, env, session, "/admin/users/new", newUserForm(token, "valid@test.com"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 for a valid-token create, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
	if loc := resp.Header.Get("Location"); loc != "/admin/users" {
		t.Errorf("expected redirect to /admin/users, got %q", loc)
	}
}

// TestAdminCSRF_RejectsTokenReuse: tokens are single-use. A second POST with the
// same token must be rejected with 403 (captured-token replay defense).
func TestAdminCSRF_RejectsTokenReuse(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	token := fetchCSRFToken(t, env, session, "/admin/users/new")

	first := postAdminForm(t, env, session, "/admin/users/new", newUserForm(token, "reuse1@test.com"))
	first.Body.Close()
	if first.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected first POST to succeed (303), got %d", first.StatusCode)
	}

	second := postAdminForm(t, env, session, "/admin/users/new", newUserForm(token, "reuse2@test.com"))
	second.Body.Close()
	if second.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 on token reuse, got %d", second.StatusCode)
	}
}

// TestAdminCSRF_ConcurrentTabsBothWork: two tabs each fetch their own token;
// both remain valid until individually consumed (multiple valid tokens per
// session — no "second tab invalidates the first").
func TestAdminCSRF_ConcurrentTabsBothWork(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	tokenA := fetchCSRFToken(t, env, session, "/admin/users/new")
	tokenB := fetchCSRFToken(t, env, session, "/admin/users/new")
	if tokenA == tokenB {
		t.Fatal("expected two distinct tokens for two form loads")
	}

	respA := postAdminForm(t, env, session, "/admin/users/new", newUserForm(tokenA, "taba@test.com"))
	respA.Body.Close()
	if respA.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected tab A POST to succeed (303), got %d", respA.StatusCode)
	}

	respB := postAdminForm(t, env, session, "/admin/users/new", newUserForm(tokenB, "tabb@test.com"))
	respB.Body.Close()
	if respB.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected tab B POST to still succeed (303) after tab A consumed its own token, got %d", respB.StatusCode)
	}
}

// TestAdminCSRF_FailurePathReMintsToken: when a create fails on business logic
// (duplicate email), the re-rendered form must carry a FRESH usable token —
// otherwise the retry would 403 because the original token was already consumed.
func TestAdminCSRF_FailurePathReMintsToken(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	// admin@test.com already exists → CreateUser fails and re-renders the form.
	token := fetchCSRFToken(t, env, session, "/admin/users/new")
	resp := postAdminForm(t, env, session, "/admin/users/new", newUserForm(token, "admin@test.com"))
	body := readBody(t, resp)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for duplicate-email create, got %d: %s", resp.StatusCode, body)
	}
	m := csrfTokenRE.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("failure re-render did not include a fresh csrf_token: %s", body)
	}
	// The re-minted token must itself be usable on a subsequent valid submit.
	resp2 := postAdminForm(t, env, session, "/admin/users/new", newUserForm(m[1], "freshretry@test.com"))
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected the re-minted token to work on retry (303), got %d", resp2.StatusCode)
	}
}

// TestAdminLogin_RotatesSessionID: a session cookie planted before login must be
// replaced by a DIFFERENT session id on successful auth (session-fixation
// defense).
func TestAdminLogin_RotatesSessionID(t *testing.T) {
	env := setupAdminServer(t)

	const planted = "attacker-planted-session-id"
	form := url.Values{"email": {"admin@test.com"}, "password": {"adminpass1"}}
	req, _ := http.NewRequest(http.MethodPost, env.srv.URL+"/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: planted})

	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("POST /admin/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 from login, got %d", resp.StatusCode)
	}

	var newSession string
	for _, c := range resp.Cookies() {
		if c.Name == "admin_session" {
			newSession = c.Value
		}
	}
	if newSession == "" {
		t.Fatal("login did not set a new admin_session cookie")
	}
	if newSession == planted {
		t.Fatal("session id was not rotated at login — session-fixation exposure")
	}

	// The planted session must not be usable afterward (it was cleared).
	if _, ok := env.sessions.Get(planted); ok {
		t.Error("planted session id should have been discarded at login")
	}
}
