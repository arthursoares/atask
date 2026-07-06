package api_test

// Task 14: web admin UI. These tests run the admin routes on PocketBase's real
// router served over a real TCP connection (same harness style as
// pb_router_bridge_test.go / auth_test.go), with a real auth.PBAdapter over
// tests.NewTestApp() as the AuthProvider. The admin CSRF + session stores are
// constructed here and passed through RoutesDeps so the test can plant sessions
// directly to exercise requireAdmin's role check in isolation.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/atask/atask/internal/api"
	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/config"
	"github.com/atask/atask/internal/event"
	"github.com/atask/atask/internal/service"
	"github.com/atask/atask/internal/store"
)

// sessionSetter is the subset of the (unexported) *sessionStore the admin tests
// need. api_test cannot name the unexported type, but *sessionStore satisfies
// this interface, so setupAdminServer can hand it back for direct planting.
type sessionSetter interface {
	Set(sessionID, userID string)
	Get(sessionID string) (string, bool)
	Delete(sessionID string)
}

// adminEnv bundles everything an admin test needs to drive the server.
type adminEnv struct {
	srv      *httptest.Server
	sessions sessionSetter
	auth     auth.AuthProvider
}

// setupAdminServer wires the admin routes onto PocketBase's real router over a
// real TCP listener and seeds a single admin account (admin@test.com /
// adminpass1). Returns the server, the shared session store (for planting
// sessions), and the AuthProvider (for creating additional users).
func setupAdminServer(t *testing.T) adminEnv {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	if err := auth.EnsureUserFields(app); err != nil {
		t.Fatalf("auth.EnsureUserFields: %v", err)
	}

	adapter := auth.NewPBAdapterFromApp(app)
	if _, err := adapter.CreateUser("admin@test.com", "adminpass1", "Admin", "admin"); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	pbRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("apis.NewRouter: %v", err)
	}

	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("store.NewDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	es := event.NewEventStore(db)
	bus := event.NewBus()
	sessions := api.NewSessionStore()

	se := &core.ServeEvent{App: app, Router: pbRouter}
	api.RegisterRoutes(se, api.RoutesDeps{
		DB:            db,
		AuthProvider:  adapter,
		AuthService:   service.NewAuthService(db, "test-secret"),
		Config:        &config.Config{}, // BaseURL "" → Secure=false, cookies flow over http
		EventStore:    es,
		Bus:           bus,
		StreamManager: event.NewStreamManager(bus),
		TaskSvc:       service.NewTaskService(db, es, bus),
		ProjectSvc:    service.NewProjectService(db, es, bus),
		AreaSvc:       service.NewAreaService(db, es, bus),
		SectionSvc:    service.NewSectionService(db, es, bus),
		TagSvc:        service.NewTagService(db, es, bus),
		LocationSvc:   service.NewLocationService(db, es, bus),
		ChecklistSvc:  service.NewChecklistService(db, es, bus),
		ActivitySvc:   service.NewActivityService(db, es, bus),
		CSRFStore:     api.NewCSRFStore(),
		SessionStore:  sessions,
	})

	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return adminEnv{srv: srv, sessions: sessions, auth: adapter}
}

// noRedirectClient returns an http.Client that surfaces 3xx responses instead
// of following them, so tests can assert on redirect status + Set-Cookie.
func noRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// loginAsAdmin performs the login POST and returns the admin_session cookie.
func loginAsAdmin(t *testing.T, env adminEnv, email, password string) *http.Cookie {
	t.Helper()
	form := url.Values{"email": {email}, "password": {password}}
	resp, err := noRedirectClient().Post(
		env.srv.URL+"/admin/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("POST /admin/login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 from login, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
	for _, c := range resp.Cookies() {
		if c.Name == "admin_session" && c.Value != "" {
			return c
		}
	}
	t.Fatal("login did not set an admin_session cookie")
	return nil
}

var csrfTokenRE = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)

// fetchCSRFToken GETs a form page with the given session cookie and extracts the
// hidden csrf_token value from the returned HTML.
func fetchCSRFToken(t *testing.T, env adminEnv, session *http.Cookie, path string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, env.srv.URL+path, nil)
	req.AddCookie(session)
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	body := readBody(t, resp)
	m := csrfTokenRE.FindStringSubmatch(body)
	if m == nil {
		t.Fatalf("no csrf_token found in %s response: %s", path, body)
	}
	return m[1]
}

// postAdminForm POSTs a urlencoded form with the session cookie, not following
// redirects.
func postAdminForm(t *testing.T, env adminEnv, session *http.Cookie, path string, form url.Values) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, env.srv.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if session != nil {
		req.AddCookie(session)
	}
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// TestRequireAdmin_AdminGets200 confirms a logged-in admin reaches the dashboard.
func TestRequireAdmin_AdminGets200(t *testing.T) {
	env := setupAdminServer(t)
	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")

	req, _ := http.NewRequest(http.MethodGet, env.srv.URL+"/admin/", nil)
	req.AddCookie(session)
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("GET /admin/: %v", err)
	}
	defer resp.Body.Close()
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for admin dashboard, got %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "admin@test.com") {
		t.Errorf("expected the dashboard to list the admin user, got: %s", body)
	}
}

// TestRequireAdmin_NonAdminGets403 plants a session for a non-admin user
// directly in the session store and confirms requireAdmin's role check rejects
// it with 403 (not a redirect — the session is valid, the role is not).
func TestRequireAdmin_NonAdminGets403(t *testing.T) {
	env := setupAdminServer(t)

	regular, err := env.auth.CreateUser("regular@test.com", "regularpw1", "Reg", "user")
	if err != nil {
		t.Fatalf("create regular user: %v", err)
	}
	env.sessions.Set("planted-nonadmin-session", regular.ID)

	req, _ := http.NewRequest(http.MethodGet, env.srv.URL+"/admin/", nil)
	req.AddCookie(&http.Cookie{Name: "admin_session", Value: "planted-nonadmin-session"})
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("GET /admin/: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for a non-admin session, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

// TestRequireAdmin_NoSessionRedirects confirms an unauthenticated request to a
// protected route is redirected to the login page.
func TestRequireAdmin_NoSessionRedirects(t *testing.T) {
	env := setupAdminServer(t)

	resp, err := noRedirectClient().Get(env.srv.URL + "/admin/")
	if err != nil {
		t.Fatalf("GET /admin/ (no session): %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect without a session, got %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/admin/login" {
		t.Errorf("expected redirect to /admin/login, got %q", loc)
	}
}

// TestAdminLoginPage_HasForm confirms the public login page renders a form.
func TestAdminLoginPage_HasForm(t *testing.T) {
	env := setupAdminServer(t)
	resp, err := http.Get(env.srv.URL + "/admin/login")
	if err != nil {
		t.Fatalf("GET /admin/login: %v", err)
	}
	defer resp.Body.Close()
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(body, `action="/admin/login"`) || !strings.Contains(body, `name="password"`) {
		t.Errorf("login page missing expected form: %s", body)
	}
}

// TestAdminLogin_RejectsNonAdmin confirms a valid non-admin credential cannot
// establish an admin session.
func TestAdminLogin_RejectsNonAdmin(t *testing.T) {
	env := setupAdminServer(t)
	if _, err := env.auth.CreateUser("reg2@test.com", "reg2pass12", "Reg2", "user"); err != nil {
		t.Fatalf("create user: %v", err)
	}
	form := url.Values{"email": {"reg2@test.com"}, "password": {"reg2pass12"}}
	resp := postAdminForm(t, env, nil, "/admin/login", form)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin login, got %d", resp.StatusCode)
	}
	for _, c := range resp.Cookies() {
		if c.Name == "admin_session" && c.Value != "" {
			t.Fatal("non-admin login must not set a session cookie")
		}
	}
}
