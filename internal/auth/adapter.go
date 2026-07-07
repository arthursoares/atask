package auth

import "time"

// User represents an authenticated user.
type User struct {
	ID        string
	Email     string
	Name      string
	Role      string // "user" or "admin"
	Disabled  bool
	AvatarURL string
	Verified  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AuthProvider abstracts the identity backend. The PocketBase implementation
// is the only concrete implementation, but the interface isolates upgrade risk.
type AuthProvider interface {
	// Token validation
	ValidateToken(token string) (userID string, err error)

	// User CRUD
	CreateUser(email, password, name, role string) (*User, error)
	FindUserByID(id string) (*User, error)
	FindUserByEmail(email string) (*User, error)
	UpdateUser(id string, updates map[string]any) error
	DisableUser(id string) error
	EnableUser(id string) error
	DeleteUser(id string) error
	ListUsers(filter string, page, perPage int) ([]*User, int, error)

	// Auth
	AuthWithPassword(email, password string) (token string, user *User, err error)
	RefreshToken(token string) (newToken string, err error)

	// Provider discovery
	EnabledProviders() []string
}
