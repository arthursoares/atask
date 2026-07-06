package auth

import (
	"fmt"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

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
		return "", nil, fmt.Errorf("user not found: %w", err)
	}
	if !record.ValidatePassword(password) {
		return "", nil, fmt.Errorf("invalid password")
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
		if err == nil {
			total = len(all)
		}
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
