package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// User is the domain type returned by AuthService — no password hash.
type User struct {
	ID    string
	Email string
	Name  string
}

// errLegacyAuthRemoved is returned by AuthService methods that depended on the local
// `users` table, which was dropped in migration 006 (legacy cleanup). Identity now
// lives in PocketBase; Task 12 replaces these methods with an AuthProvider-backed
// implementation.
var errLegacyAuthRemoved = errors.New("legacy auth removed; use AuthProvider")

// APIKey is the domain type for API key metadata.
type APIKey struct {
	ID          string
	UserID      string
	Name        string
	Permissions []string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
}

// AuthService handles user authentication and API key management.
type AuthService struct {
	queries   *sqlc.Queries
	jwtSecret []byte
}

// NewAuthService constructs an AuthService backed by the given DB.
func NewAuthService(db *store.DB, jwtSecret string) *AuthService {
	return &AuthService{
		queries:   sqlc.New(db.DB),
		jwtSecret: []byte(jwtSecret),
	}
}

// apiKeyFromRow converts a sqlc ApiKey row to the domain APIKey.
func apiKeyFromRow(row sqlc.ApiKey) *APIKey {
	k := &APIKey{
		ID:          row.ID,
		UserID:      row.UserID,
		Permissions: []string{},
	}
	if row.Name.Valid {
		k.Name = row.Name.String
	}
	if row.CreatedAt.Valid {
		k.CreatedAt = row.CreatedAt.Time
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		k.LastUsedAt = &t
	}
	return k
}

// CreateUser is stubbed out; legacy local user creation was removed in migration 006
// (the `users` table was dropped). Task 12 rewrites registration via AuthProvider.
func (s *AuthService) CreateUser(ctx context.Context, email, password, name string) (*User, error) {
	return nil, errLegacyAuthRemoved
}

// Login is stubbed out; legacy local user creation was removed in migration 006
// (the `users` table was dropped). Task 12 rewrites login via AuthProvider.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	return "", errLegacyAuthRemoved
}

// ValidateToken parses and validates the JWT, returning the user ID from the subject claim.
func (s *AuthService) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token claims")
	}

	return claims.Subject, nil
}

// GetUser is stubbed out; legacy local user lookup was removed in migration 006
// (the `users` table was dropped). Task 12 rewrites user lookup via AuthProvider.
func (s *AuthService) GetUser(ctx context.Context, id string) (*User, error) {
	return nil, errLegacyAuthRemoved
}

// UpdateUser is stubbed out; legacy local user updates were removed in migration 006
// (the `users` table was dropped). Task 12 rewrites profile updates via AuthProvider.
func (s *AuthService) UpdateUser(ctx context.Context, id, name string) error {
	return errLegacyAuthRemoved
}

// hashAPIKey returns the hex-encoded SHA256 of the given key.
func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CreateAPIKey generates a random 32-byte hex key, persists its SHA256 hash, and returns
// the plaintext key (only once) along with the key metadata.
func (s *AuthService) CreateAPIKey(ctx context.Context, userID, name string) (string, *APIKey, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate api key: %w", err)
	}
	plainKey := hex.EncodeToString(raw)
	keyHash := hashAPIKey(plainKey)

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateAPIKey(ctx, sqlc.CreateAPIKeyParams{
		ID:          id,
		UserID:      userID,
		Name:        sql.NullString{String: name, Valid: true},
		KeyHash:     sql.NullString{String: keyHash, Valid: true},
		Permissions: "[]",
		Scope:       "read_write",
		ExpiresAt:   sql.NullTime{},
		CreatedAt:   sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return "", nil, fmt.Errorf("create api key: %w", err)
	}

	return plainKey, apiKeyFromRow(row), nil
}

// ValidateAPIKey looks up the key by its SHA256 hash, updates last_used_at, and returns the
// owning user ID, the key ID (for actor attribution), and the key's scope (e.g. "read_write",
// "read_only"). Rows created before migration 006 default to "read_write".
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (userID, keyID, scope string, err error) {
	keyHash := hashAPIKey(key)

	row, err := s.queries.GetAPIKeyByHash(ctx, sql.NullString{String: keyHash, Valid: true})
	if err != nil {
		return "", "", "", errors.New("invalid api key")
	}

	// Belt-and-suspenders expiry check: GetAPIKeyByHash's SQL predicate
	// (`expires_at IS NULL OR expires_at > datetime('now')`, Task 1.5) is
	// meant to already exclude expired rows, but the modernc.org/sqlite
	// driver serializes a bound time.Time in a form SQLite's own date
	// functions cannot reliably parse back out of the column (confirmed via
	// isolated repro: datetime(expires_at)/julianday(expires_at) both come
	// back NULL for a value written as sql.NullTime), which silently
	// defeats that string/date comparison. Re-check expiry here in Go, where
	// row.ExpiresAt has already been decoded by database/sql into a real
	// time.Time, so an expired key is rejected regardless of the SQL
	// predicate's driver-dependent behavior.
	if row.ExpiresAt.Valid && !row.ExpiresAt.Time.After(timeNow()) {
		return "", "", "", errors.New("invalid api key")
	}

	now := timeNow()
	if err := s.queries.UpdateAPIKeyLastUsed(ctx, sqlc.UpdateAPIKeyLastUsedParams{
		LastUsedAt: sql.NullTime{Time: now, Valid: true},
		ID:         row.ID,
	}); err != nil {
		return "", "", "", fmt.Errorf("update last_used_at: %w", err)
	}

	return row.UserID, row.ID, row.Scope, nil
}

// ListAPIKeys returns the API key metadata for the given user (no hashes).
func (s *AuthService) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	rows, err := s.queries.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	keys := make([]*APIKey, len(rows))
	for i, row := range rows {
		keys[i] = apiKeyFromRow(row)
	}
	return keys, nil
}

// UpdateAPIKeyName renames an API key, verifying that it belongs to the given user.
func (s *AuthService) UpdateAPIKeyName(ctx context.Context, id, userID, name string) error {
	// Verify ownership first
	rows, err := s.queries.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list api keys: %w", err)
	}
	found := false
	for _, r := range rows {
		if r.ID == id {
			found = true
			break
		}
	}
	if !found {
		return errors.New("api key not found or not owned by user")
	}

	_, err = s.queries.UpdateAPIKeyName(ctx, sqlc.UpdateAPIKeyNameParams{
		Name: sql.NullString{String: name, Valid: true},
		ID:   id,
	})
	if err != nil {
		return fmt.Errorf("update api key name: %w", err)
	}
	return nil
}

// DeleteAPIKey removes an API key, verifying that it belongs to the given user.
func (s *AuthService) DeleteAPIKey(ctx context.Context, id, userID string) error {
	// Verify ownership first
	rows, err := s.queries.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list api keys: %w", err)
	}
	found := false
	for _, r := range rows {
		if r.ID == id {
			found = true
			break
		}
	}
	if !found {
		return errors.New("api key not found or not owned by user")
	}

	return s.queries.DeleteAPIKey(ctx, id)
}
