package api_test

// This file exercises RegisterRoutes/bridge() through PocketBase's *real*
// router (apis.NewRouter + Router.BuildMux) served over a *real* TCP
// connection (httptest.NewServer), the one request path every other test
// server in this package deliberately bypasses by registering handlers
// directly on a bare *http.ServeMux and driving them with
// httptest.NewRequest+ResponseRecorder (see decode_integration_test.go's
// setupTaskAndAuthTestServer, middleware_test.go's buildFullTestMux, etc.).
//
// Both shortcuts matter, not just the PB-router one: reproducing the bug
// also requires a genuine network round trip. net/http's server-side request
// body (net/http.(*body).readLocked, see transfer.go) deliberately bundles
// io.EOF together with the final chunk of data on the read that exhausts a
// known Content-Length ("If we can return an EOF here along with the read
// data, do so ... helps the HTTP transport code recycle its connection
// earlier"). That bundled EOF is what makes PocketBase's
// RereadableReadCloser.Read (tools/router/router.go) rewind itself *during*
// DecodeJSON's *first* dec.Decode(dst) call — before DecodeJSON's second,
// EOF-checking dec.Decode(&struct{}{}) ever runs. httptest.NewRequest backed
// by a bytes.Reader/bytes.Buffer does NOT bundle EOF this way (bytes.Reader
// only ever returns io.EOF on a separate, later call once exhausted), so
// driving the mux directly via httptest.NewRecorder cannot reproduce this
// bug even through the real PB router — it must go over a real connection.
//
// The upshot: PocketBase's router wraps every request body in a
// RereadableReadCloser whose Read auto-"rewinds" back to the start on
// io.EOF (so PB's own multi-stage hook pipeline can read a body more than
// once). DecodeJSON's single-JSON-object guard (internal/api/response.go)
// does a second dec.Decode(&struct{}{}) expecting a clean io.EOF; through
// the rewound body that second decode silently re-reads the same JSON
// object and DecodeJSON mistakes it for trailing data. Every valid
// body-carrying request — POST /auth/login, POST /auth/register,
// POST /tasks, every PATCH — failed with 400 "request body must contain a
// single JSON object" the moment it went through PocketBase's real router
// over a real connection, even though the handler-level tests (hitting the
// mux directly with httptest.NewRecorder) stayed green. bridge() (routes.go)
// now drains the body once and installs a plain io.NopCloser before invoking
// the wrapped handler, undoing PB's rewind.

import (
	"io"
	"net/http"
	"net/http/httptest"
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

// startRealPBServer wires atask's domain routes onto PocketBase's actual
// router exactly the way cmd/atask/main.go does in production: build the PB
// router (apis.NewRouter), call api.RegisterRoutes on a *core.ServeEvent
// wrapping it, compile the router into an http.Handler via
// Router.BuildMux(), and serve it over a real TCP listener via
// httptest.NewServer. Both the real PB router AND the real network round
// trip are required to reproduce the Critical bug — see the file-level doc
// comment for why a bare mux + httptest.NewRecorder cannot.
func startRealPBServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv, _ := startRealPBServerWithApp(t)
	return srv
}

// startRealPBServerWithApp is startRealPBServer, additionally returning the
// underlying *tests.TestApp so callers can mint tokens directly against
// PocketBase's seeded auth collections (e.g. a _superusers token — see
// TestAuth_Refresh_RejectsSuperuserToken in auth_test.go) rather than only
// through the HTTP-facing register/login flow.
//
// RegistrationOpen defaults to true here so every pre-Task-17 test in this
// package — which all assume POST /auth/register succeeds openly — keeps
// working unchanged. Task 17's invite-flow tests need the opposite
// (RegistrationOpen: false, invite required) and use
// startRealPBServerWithConfig directly instead.
func startRealPBServerWithApp(t *testing.T) (*httptest.Server, *tests.TestApp) {
	t.Helper()
	srv, app, _ := startRealPBServerWithConfig(t, &config.Config{RegistrationOpen: true})
	return srv, app
}

// startRealPBServerWithConfig is startRealPBServerWithApp, additionally
// taking the *config.Config to wire as RoutesDeps.Config and returning the
// underlying *store.DB — so callers can exercise RegistrationOpen=false
// (invite-gated registration, invite_test.go), a non-default BaseURL (invite
// URL assertions), or reach into the invites table directly (e.g. to
// backdate expires_at for an expired-invite test) without touching the
// common-case helper every other test in this package relies on.
func startRealPBServerWithConfig(t *testing.T, cfg *config.Config) (*httptest.Server, *tests.TestApp, *store.DB) {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("tests.NewTestApp: %v", err)
	}
	t.Cleanup(app.Cleanup)

	// Give the test app's `users` collection the role/disabled fields the
	// real boot path adds (cmd/atask/main.go), so a fresh read after Save
	// (e.g. AuthWithPassword's FindAuthRecordByEmail during Login) doesn't
	// silently lose them — see auth.EnsureUserFields's doc comment.
	if err := auth.EnsureUserFields(app); err != nil {
		t.Fatalf("auth.EnsureUserFields: %v", err)
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

	se := &core.ServeEvent{App: app, Router: pbRouter}
	api.RegisterRoutes(se, api.RoutesDeps{
		DB:            db,
		AuthProvider:  auth.NewPBAdapterFromApp(app),
		AuthService:   service.NewAuthService(db, "test-secret"),
		Config:        cfg,
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
	})

	mux, err := pbRouter.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, app, db
}

// TestRealPBRouter_ValidBody_NotRejectedAsTrailingData is the regression
// guard for the Critical bug: POST a single, valid JSON object to
// /auth/register over a real connection through PocketBase's real router,
// and assert DecodeJSON does NOT mistake PB's rewound body for a second
// (trailing) JSON object. Without the bridge() fix this fails with 400
// "request body must contain a single JSON object" for every request —
// reproduced live via
// `curl -X POST localhost:8091/auth/register -d '{"email":"probe@test.com","password":"secret123","name":"Probe"}'`.
//
// /auth/register now creates a real account via the AuthProvider (Task 12),
// so the *correct* post-fix response is 201 with the created user's profile —
// a real auth-layer response, not the transport-layer 400 the bug produced.
// That distinction is exactly the point: the bug fired before the handler
// ever got to run business logic.
func TestRealPBRouter_ValidBody_NotRejectedAsTrailingData(t *testing.T) {
	srv := startRealPBServer(t)

	body := `{"email":"probe@test.com","password":"secret123","name":"Probe"}`
	resp, err := http.Post(srv.URL+"/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer resp.Body.Close()

	respBody := readBody(t, resp)

	if resp.StatusCode == http.StatusBadRequest && strings.Contains(respBody, "single JSON object") {
		t.Fatalf("regression: valid single-object body rejected as trailing data through PocketBase's real router over a real connection (bridge() body-rewind bug): %d %s", resp.StatusCode, respBody)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 (real AuthProvider-backed registration) once the transport-layer bug is fixed, got %d: %s", resp.StatusCode, respBody)
	}
	if !strings.Contains(respBody, `"email":"probe@test.com"`) {
		t.Errorf("expected the created user's email in the response body, got: %s", respBody)
	}
}

// TestRealPBRouter_MalformedTrailingJSON_StillRejected confirms the fix
// doesn't neuter DecodeJSON's real trailing-data guard: a body with two
// concatenated JSON objects must still be rejected with 400 over a real
// connection through the real PB router.
func TestRealPBRouter_MalformedTrailingJSON_StillRejected(t *testing.T) {
	srv := startRealPBServer(t)

	body := `{"email":"a@example.com","password":"secret123","name":"A"}{"email":"b@example.com"}`
	resp, err := http.Post(srv.URL+"/auth/register", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /auth/register: %v", err)
	}
	defer resp.Body.Close()

	respBody := readBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for genuinely malformed (two-object) body, got %d: %s", resp.StatusCode, respBody)
	}
	if !strings.Contains(respBody, "single JSON object") {
		t.Errorf("expected 'single JSON object' error message, got: %s", respBody)
	}
}

// TestRealPBRouter_HealthGet_Unaffected confirms bufferBody's no-op path for
// bodyless GET requests works through the real PB router over a real
// connection too (SSE's stream route and every other GET/HEAD/DELETE-without-
// body share this path).
func TestRealPBRouter_HealthGet_Unaffected(t *testing.T) {
	srv := startRealPBServer(t)

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /health, got %d: %s", resp.StatusCode, readBody(t, resp))
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		t.Fatalf("read response body: %v", err)
	}
	return buf.String()
}
