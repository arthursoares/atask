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
	return &PBAdapter{app: app}
}

func (a *PBAdapter) ValidateToken(token string) (string, error) {
	record, err := a.app.FindAuthRecordByToken(token, core.TokenTypeAuth)
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}
	// Only accept tokens belonging to the application users collection. A
	// _superusers (admin) auth token authenticates against a different
	// collection and must NOT be accepted as an application user identity.
	if record.Collection() == nil || record.Collection().Name != usersCollection {
		return "", errors.New("invalid token: not a users-collection token")
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
	// Populated from config in the wiring layer (Task 11+); nil here.
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
