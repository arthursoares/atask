package api_test

// Task 14: web admin UI. These tests run the admin routes on PocketBase's real
// router served over a real TCP connection (same harness style as
// pb_router_bridge_test.go / auth_test.go), with a real auth.PBAdapter over
// tests.NewTestApp() as the AuthProvider. The admin CSRF + session stores are
// constructed here and passed through RoutesDeps so the test can plant sessions
// directly to exercise requireAdmin's role check in isolation.

import (
	"context"
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
	// db is the domain SQLite database RegisterRoutes wired the dashboard to
	// (Task 22: lets tests seed orphaned rows ahead of a dashboard GET).
	db *store.DB
}

// setupAdminServer wires the admin routes onto PocketBase's real router over a
// real TCP listener and seeds a single admin account (admin@test.com /
// adminpass1). Returns the server, the shared session store (for planting
// sessions), the AuthProvider (for creating additional users), and the domain
// DB (for seeding rows directly, e.g. orphaned data).
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

	return adminEnv{srv: srv, sessions: sessions, auth: adapter, db: db}
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

// TestAdminDashboard_ShowsOrphanBanner confirms the Task 22 orphan-data
// warning banner renders on the dashboard once a pre-multi-user row
// (user_id = '') exists.
func TestAdminDashboard_ShowsOrphanBanner(t *testing.T) {
	env := setupAdminServer(t)

	_, err := env.db.DB.Exec(`INSERT INTO tasks (id, user_id, title, "index", today_index, created_at, updated_at) VALUES ('orphan-1', '', 'orphan task', 0, 0, datetime('now'), datetime('now'))`)
	if err != nil {
		t.Fatalf("insert orphan task: %v", err)
	}

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
	if !strings.Contains(body, "orphaned rows detected") {
		t.Errorf("expected dashboard to show the orphan banner, got: %s", body)
	}
}

// TestAdminDashboard_ShowsStatsAndRecentEvent seeds two tasks (and their
// task.created domain_events) for a real member account, via the same
// TaskService the JSON API uses, then confirms the dashboard's stat tiles,
// growth chart, recent-activity table, and per-user row all reflect that
// seeded data -- not just a 200.
func TestAdminDashboard_ShowsStatsAndRecentEvent(t *testing.T) {
	env := setupAdminServer(t)

	member, err := env.auth.CreateUser("member@test.com", "memberpass1", "Member", "user")
	if err != nil {
		t.Fatalf("create member user: %v", err)
	}

	es := event.NewEventStore(env.db)
	bus := event.NewBus()
	taskSvc := service.NewTaskService(env.db, es, bus)
	ctx := context.Background()
	if _, err := taskSvc.Create(ctx, member.ID, "Stats task one", member.ID); err != nil {
		t.Fatalf("create task 1: %v", err)
	}
	if _, err := taskSvc.Create(ctx, member.ID, "Stats task two", member.ID); err != nil {
		t.Fatalf("create task 2: %v", err)
	}

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

	// Stat tile: system-wide task count reflects the two seeded tasks (no
	// other account has any).
	if !strings.Contains(body, `<p class="stat">2</p><p class="muted">tasks</p>`) {
		t.Errorf("expected the tasks stat tile to show 2, got: %s", body)
	}

	// Recent-activity table: the seeded task.created events must appear,
	// attributed to the member as actor.
	if !strings.Contains(body, "task.created") {
		t.Errorf("expected recent-activity table to show task.created, got: %s", body)
	}
	if !strings.Contains(body, "<td>"+member.ID+"</td>") {
		t.Errorf("expected recent-activity table to attribute an event to actor %s, got: %s", member.ID, body)
	}

	// Users table: the member's row must show 2 items and a non-"never"
	// last-active value (they just generated events).
	rowRE := regexp.MustCompile(`(?s)<tr>\s*<td><a href="/admin/users/` + regexp.QuoteMeta(member.ID) + `">.*?</tr>`)
	row := rowRE.FindString(body)
	if row == "" {
		t.Fatalf("expected a user-table row for %s, got: %s", member.ID, body)
	}
	if !strings.Contains(row, "<td>2</td>") {
		t.Errorf("expected member row to show 2 items, got row: %s", row)
	}
	if strings.Contains(row, ">never<") {
		t.Errorf("expected member row to show a last-active time, got row: %s", row)
	}
}

// TestAdminUserEdit_ShowsOverview seeds three tasks for a member account and
// confirms the user-detail page's overview block (join date, count tiles,
// total events, per-user recent-activity table) renders the seeded values.
func TestAdminUserEdit_ShowsOverview(t *testing.T) {
	env := setupAdminServer(t)

	member, err := env.auth.CreateUser("overview@test.com", "overviewpw1", "Overview", "user")
	if err != nil {
		t.Fatalf("create member user: %v", err)
	}

	es := event.NewEventStore(env.db)
	bus := event.NewBus()
	taskSvc := service.NewTaskService(env.db, es, bus)
	ctx := context.Background()
	for _, title := range []string{"Overview task one", "Overview task two", "Overview task three"} {
		if _, err := taskSvc.Create(ctx, member.ID, title, member.ID); err != nil {
			t.Fatalf("create task %q: %v", title, err)
		}
	}

	session := loginAsAdmin(t, env, "admin@test.com", "adminpass1")
	req, _ := http.NewRequest(http.MethodGet, env.srv.URL+"/admin/users/"+member.ID, nil)
	req.AddCookie(session)
	resp, err := noRedirectClient().Do(req)
	if err != nil {
		t.Fatalf("GET /admin/users/%s: %v", member.ID, err)
	}
	defer resp.Body.Close()
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for user-edit page, got %d: %s", resp.StatusCode, body)
	}

	if !strings.Contains(body, "Joined "+member.CreatedAt.Format("2006-01-02")) {
		t.Errorf("expected overview to show the join date, got: %s", body)
	}
	if !strings.Contains(body, `<p class="stat">3</p><p class="muted">tasks</p>`) {
		t.Errorf("expected overview to show 3 tasks, got: %s", body)
	}
	if !strings.Contains(body, "3 total events") {
		t.Errorf("expected overview to show 3 total events, got: %s", body)
	}
	if !strings.Contains(body, "task.created") {
		t.Errorf("expected the per-user recent-activity table to show task.created, got: %s", body)
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
