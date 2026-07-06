package auth

import (
	"errors"
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"
)

// usersCollection is the name of the PocketBase auth collection that holds
// application users. Tokens issued for any other auth collection (notably
// _superusers) must not validate as an application user.
const usersCollection = "users"

// dummyBcryptHash is a valid bcrypt hash of a random string, used to perform a
// constant-time-ish password comparison on the user-not-found path so that
// AuthWithPassword does not leak account existence via response timing.
//
// Review note: the "$2a$10$..." prefix bakes in bcrypt cost 10, matching
// PocketBase's default cost for the `users` auth collection's password field.
// If the users collection's password cost is ever reconfigured away from the
// default, regenerate this hash at the new cost so the dummy comparison's
// timing still matches a real lookup.
const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// errInvalidCredentials is the single error returned for both "user not found"
// and "wrong password" so callers cannot distinguish the two (no user
// enumeration oracle).
var errInvalidCredentials = errors.New("invalid credentials")

// Compile-time assertion that PBAdapter satisfies the AuthProvider interface.
var _ AuthProvider = (*PBAdapter)(nil)

// PBAdapter implements AuthProvider using PocketBase's Go API (v0.39.x).
type PBAdapter struct {
	app core.App
}

// NewPBAdapter builds an adapter around a running PocketBase app.
// It accepts *pocketbase.PocketBase (which embeds core.App).
func NewPBAdapter(app *pocketbase.PocketBase) *PBAdapter {
	return NewPBAdapterFromApp(app)
}

// NewPBAdapterFromApp builds an adapter around any core.App. This is the same
// construction NewPBAdapter delegates to, exposed for callers that only have
// a core.App — notably *tests.TestApp (github.com/pocketbase/pocketbase/tests),
// which embeds *core.BaseApp rather than *pocketbase.PocketBase. Task 12's
// internal/api tests use this to exercise the real AuthProvider end to end
// (register/login/refresh/me) against a real PocketBase test app.
func NewPBAdapterFromApp(app core.App) *PBAdapter {
	return &PBAdapter{app: app}
}

// EnsureUserFields adds the custom `role` (text) and `disabled` (bool) fields
// to PocketBase's `users` auth collection if they are missing (name + avatar
// ship on the collection by default). Idempotent: safe to call on every serve.
//
// This matters beyond the initial Save: Record.Set on a key that has no
// matching collection field only stores the value in memory (see
// Record.Set/SetIfFieldExists in pocketbase/core) — it is never persisted as
// a DB column. Without this field present on the collection, CreateUser's
// role/disabled values would look correct on the record returned from the
// same Save call, but silently vanish on any subsequent fresh read (e.g. the
// FindAuthRecordByEmail a later Login performs). The production boot path
// (cmd/atask/main.go) calls this once per serve; tests that spin up their own
// PocketBase test app (internal/api's auth tests) call it too for parity.
func EnsureUserFields(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}

	changed := false
	if collection.Fields.GetByName("role") == nil {
		collection.Fields.Add(&core.TextField{Name: "role"})
		changed = true
	}
	if collection.Fields.GetByName("disabled") == nil {
		collection.Fields.Add(&core.BoolField{Name: "disabled"})
		changed = true
	}
	if !changed {
		return nil
	}
	return app.Save(collection)
}

// errNotUsersCollectionToken is returned whenever a resolved auth token
// belongs to an auth collection other than usersCollection (notably
// _superusers). Shared by ValidateToken and RefreshToken so both reject a
// non-application token identically instead of duplicating the guard/string.
var errNotUsersCollectionToken = errors.New("invalid token: not a users-collection token")

// requireUsersCollectionRecord enforces that record belongs to the
// application users collection. A _superusers (admin) auth record
// authenticates against a different collection and must NOT be accepted as
// an application user identity — by ValidateToken (would forge an app
// identity) or by RefreshToken (would rotate/renew a superuser token).
func requireUsersCollectionRecord(record *core.Record) error {
	if record.Collection() == nil || record.Collection().Name != usersCollection {
		return errNotUsersCollectionToken
	}
	return nil
}

func (a *PBAdapter) ValidateToken(token string) (string, error) {
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	if err := requireUsersCollectionRecord(record); err != nil {
		return "", err
	}
	return record.Id, nil
}

func (a *PBAdapter) FindUserByID(id string) (*User, error) {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) FindUserByEmail(email string) (*User, error) {
	record, err := a.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) CreateUser(email, password, name, role string) (*User, error) {
	collection, err := a.app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.SetEmail(email)
	record.SetPassword(password)
	record.Set("name", name)
	record.Set("role", role)
	if err := a.app.Save(record); err != nil {
		return nil, err
	}
	return recordToUser(record), nil
}

func (a *PBAdapter) AuthWithPassword(email, password string) (string, *User, error) {
	record, err := a.app.FindAuthRecordByEmail("users", email)
	if err != nil {
		// Perform a dummy bcrypt comparison so the not-found path takes
		// comparable time to the wrong-password path, and return the same
		// error message — no account-enumeration oracle.
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		return "", nil, errInvalidCredentials
	}
	if !record.ValidatePassword(password) {
		return "", nil, errInvalidCredentials
	}
	token, err := record.NewAuthToken()
	if err != nil {
		return "", nil, err
	}
	return token, recordToUser(record), nil
}

func (a *PBAdapter) RefreshToken(token string) (string, error) {
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return "", err
	}
	// Mirror ValidateToken's collection guard: a _superusers token must not
	// be accepted here either, or POST /auth/refresh (a public route) would
	// rotate an admin token into a fresh one for any caller who obtains it.
	if err := requireUsersCollectionRecord(record); err != nil {
		return "", err
	}
	newToken, err := record.NewAuthToken()
	if err != nil {
		return "", err
	}
	return newToken, nil
}

func (a *PBAdapter) UpdateUser(id string, updates map[string]any) error {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return err
	}
	for k, v := range updates {
		record.Set(k, v)
	}
	return a.app.Save(record)
}

func (a *PBAdapter) DisableUser(id string) error {
	return a.UpdateUser(id, map[string]any{"disabled": true})
}

func (a *PBAdapter) EnableUser(id string) error {
	return a.UpdateUser(id, map[string]any{"disabled": false})
}

func (a *PBAdapter) DeleteUser(id string) error {
	record, err := a.app.FindRecordById("users", id)
	if err != nil {
		return err
	}
	return a.app.Delete(record)
}

func (a *PBAdapter) ListUsers(filter string, page, perPage int) ([]*User, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 30
	}
	records, err := a.app.FindRecordsByFilter("users", filter, "-created", perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	users := make([]*User, len(records))
	for i, r := range records {
		users[i] = recordToUser(r)
	}

	// Divergence from brief: app.CountRecords takes variadic dbx.Expression, not a
	// string filter, so it cannot count a PocketBase filter-string directly. To get
	// the total for pagination we re-run FindRecordsByFilter with limit=0 (which
	// returns ALL matching records, ignoring pagination) and take its length.
	total := len(records)
	if len(records) == perPage || page > 1 {
		all, err := a.app.FindRecordsByFilter("users", filter, "", 0, 0)
		if err != nil {
			// Do not silently return a wrong total — surface the error so the
			// caller (and its pagination) is not misled.
			return nil, 0, fmt.Errorf("count users: %w", err)
		}
		total = len(all)
	}
	return users, total, nil
}

func (a *PBAdapter) EnabledProviders() []string {
	// Vestigial: Task 12 wires GET /auth/providers directly from
	// config.Config.EnabledProviders() (which knows about configured OAuth
	// client IDs), not from the AuthProvider — PBAdapter has no config access
	// and this method is never called by the handler. Kept only to satisfy
	// the AuthProvider interface.
	return nil
}

func recordToUser(r *core.Record) *User {
	return &User{
		ID:        r.Id,
		Email:     r.Email(),
		Name:      r.GetString("name"),
		Role:      r.GetString("role"),
		Disabled:  r.GetBool("disabled"),
		AvatarURL: r.GetString("avatar"),
		Verified:  r.Verified(),
		CreatedAt: r.GetDateTime("created").Time(),
		UpdatedAt: r.GetDateTime("updated").Time(),
	}
}
