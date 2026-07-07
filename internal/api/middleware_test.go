package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/atask/atask/internal/auth"
)

// fakeAuthProvider is a minimal auth.AuthProvider stub for middleware tests.
// Only ValidateToken and FindUserByID are exercised; the rest panic if called.
type fakeAuthProvider struct {
	// validateToken maps a bearer token → resolved userID (and optional error).
	tokenUserID string
	tokenErr    error
	// users maps userID → user record returned by FindUserByID.
	users map[string]*auth.User
	// findErr, when non-nil, is returned by FindUserByID for any id.
	findErr error
}

func (f *fakeAuthProvider) ValidateToken(token string) (string, error) {
	if f.tokenErr != nil {
		return "", f.tokenErr
	}
	return f.tokenUserID, nil
}

func (f *fakeAuthProvider) FindUserByID(id string) (*auth.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	u, ok := f.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (f *fakeAuthProvider) CreateUser(email, password, name, role string) (*auth.User, error) {
	panic("not implemented")
}
func (f *fakeAuthProvider) FindUserByEmail(email string) (*auth.User, error) {
	panic("not implemented")
}
func (f *fakeAuthProvider) UpdateUser(id string, updates map[string]any) error {
	panic("not implemented")
}
func (f *fakeAuthProvider) DisableUser(id string) error { panic("not implemented") }
func (f *fakeAuthProvider) EnableUser(id string) error  { panic("not implemented") }
func (f *fakeAuthProvider) DeleteUser(id string) error  { panic("not implemented") }
func (f *fakeAuthProvider) ListUsers(filter string, page, perPage int) ([]*auth.User, int, error) {
	panic("not implemented")
}
func (f *fakeAuthProvider) AuthWithPassword(email, password string) (string, *auth.User, error) {
	panic("not implemented")
}
func (f *fakeAuthProvider) RefreshToken(token string) (string, error) { panic("not implemented") }
func (f *fakeAuthProvider) EnabledProviders() []string                { return nil }

// fakeAPIKeyValidator stubs APIKeyValidator.
type fakeAPIKeyValidator struct {
	userID string
	keyID  string
	scope  string
	err    error
	// called records whether ValidateAPIKey was invoked.
	called bool
}

func (f *fakeAPIKeyValidator) ValidateAPIKey(ctx context.Context, key string) (string, string, string, error) {
	f.called = true
	if f.err != nil {
		return "", "", "", f.err
	}
	return f.userID, f.keyID, f.scope, nil
}

// captureHandler records the userID/keyID/actor it sees and returns 200.
type captureHandler struct {
	userID string
	keyID  string
	actor  string
	served bool
}

func (c *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.served = true
	c.userID = UserIDFromContext(r.Context())
	c.keyID = KeyIDFromContext(r.Context())
	c.actor = actorFromRequest(r)
	w.WriteHeader(http.StatusOK)
}

func serve(mw func(http.Handler) http.Handler, next http.Handler, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	mw(next).ServeHTTP(rec, req)
	return rec
}

func enabledUser(id string) *auth.User { return &auth.User{ID: id, Disabled: false} }

func TestRequireAuth_BearerPath(t *testing.T) {
	ap := &fakeAuthProvider{tokenUserID: "user-1", users: map[string]*auth.User{"user-1": enabledUser("user-1")}}
	kv := &fakeAPIKeyValidator{}
	cap := &captureHandler{}

	rec := serve(requireAuth(ap, kv), cap, "Bearer sometoken")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !cap.served {
		t.Fatal("handler was not served")
	}
	if cap.userID != "user-1" {
		t.Errorf("expected userID user-1, got %q", cap.userID)
	}
	if cap.keyID != "" {
		t.Errorf("bearer path must not set keyID, got %q", cap.keyID)
	}
	// Actor attribution: bearer falls back to userID.
	if cap.actor != "user-1" {
		t.Errorf("expected actor user-1, got %q", cap.actor)
	}
	if kv.called {
		t.Error("ValidateAPIKey should not be called on the bearer path")
	}
}

func TestRequireAuth_APIKeyPath_ActorAttribution(t *testing.T) {
	ap := &fakeAuthProvider{users: map[string]*auth.User{"user-2": enabledUser("user-2")}}
	kv := &fakeAPIKeyValidator{userID: "user-2", keyID: "key-abc", scope: "read_write"}
	cap := &captureHandler{}

	rec := serve(requireAuth(ap, kv), cap, "ApiKey rawsecret")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if cap.userID != "user-2" {
		t.Errorf("expected userID user-2, got %q", cap.userID)
	}
	if cap.keyID != "key-abc" {
		t.Errorf("expected keyID key-abc, got %q", cap.keyID)
	}
	// Actor attribution prefers the key ID over the user ID.
	if cap.actor != "key-abc" {
		t.Errorf("expected actor key-abc, got %q", cap.actor)
	}
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	ap := &fakeAuthProvider{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), &captureHandler{}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_UnsupportedScheme(t *testing.T) {
	ap := &fakeAuthProvider{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), &captureHandler{}, "Basic abc")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_InvalidBearerToken(t *testing.T) {
	ap := &fakeAuthProvider{tokenErr: errors.New("bad token")}
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), cap, "Bearer bad")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served on invalid token")
	}
}

func TestRequireAuth_InvalidAPIKey(t *testing.T) {
	ap := &fakeAuthProvider{}
	kv := &fakeAPIKeyValidator{err: errors.New("invalid api key")}
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, kv), cap, "ApiKey wrong")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served on invalid api key")
	}
}

// Binding requirement #1: an empty resolved userID must be rejected with 401
// and must NOT reach FindUserByID (never become an implicit owner of the ” pool).
func TestRequireAuth_EmptyUserIDRejected_Bearer(t *testing.T) {
	ap := &fakeAuthProvider{tokenUserID: ""} // token resolves to empty subject
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), cap, "Bearer emptysub")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty userID, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served for empty userID")
	}
}

func TestRequireAuth_EmptyUserIDRejected_APIKey(t *testing.T) {
	ap := &fakeAuthProvider{}
	kv := &fakeAPIKeyValidator{userID: "", keyID: "orphan-key"}
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, kv), cap, "ApiKey emptyowner")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty userID, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served for empty userID")
	}
}

// Binding requirement #6: a disabled user is rejected with 403.
func TestRequireAuth_DisabledUserRejected(t *testing.T) {
	ap := &fakeAuthProvider{
		tokenUserID: "user-3",
		users:       map[string]*auth.User{"user-3": {ID: "user-3", Disabled: true}},
	}
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), cap, "Bearer t")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for disabled user, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served for disabled user")
	}
}

// Binding requirement #6 (orphaned key): a resolved userID whose user record is
// missing (deleted user / orphaned API key) is rejected with 401.
func TestRequireAuth_OrphanedUserRejected(t *testing.T) {
	ap := &fakeAuthProvider{tokenUserID: "ghost", users: map[string]*auth.User{}} // FindUserByID → not found
	cap := &captureHandler{}
	rec := serve(requireAuth(ap, &fakeAPIKeyValidator{}), cap, "Bearer t")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for orphaned/missing user, got %d", rec.Code)
	}
	if cap.served {
		t.Error("handler must not be served for missing user record")
	}
}
