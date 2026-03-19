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

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/atask/atask/internal/store"
	sqlc "github.com/atask/atask/internal/store/sqlc"
	"golang.org/x/crypto/bcrypt"
)

// User is the domain type returned by AuthService — no password hash.
type User struct {
	ID    string
	Email string
	Name  string
}

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

// userFromRow converts a sqlc User row to the domain User (no password hash).
func userFromRow(row sqlc.User) *User {
	return &User{
		ID:    row.ID,
		Email: row.Email,
		Name:  row.Name,
	}
}

// apiKeyFromRow converts a sqlc ApiKey row to the domain APIKey.
func apiKeyFromRow(row sqlc.ApiKey) *APIKey {
	k := &APIKey{
		ID:          row.ID,
		Permissions: []string{},
	}
	if row.UserID.Valid {
		k.UserID = row.UserID.String
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

// CreateUser hashes the password with bcrypt, persists the user, and returns the user without the password hash.
func (s *AuthService) CreateUser(ctx context.Context, email, password, name string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := timeNow()
	id := uuid.New().String()

	row, err := s.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           id,
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return userFromRow(row), nil
}

// Login verifies the user's credentials and returns a signed JWT valid for 24 hours.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	row, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	claims := jwt.RegisteredClaims{
		Subject:   row.ID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// ValidateToken parses and validates the JWT, returning the user ID from the subject claim.
func (s *AuthService) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
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

// GetUser fetches a user by ID.
func (s *AuthService) GetUser(ctx context.Context, id string) (*User, error) {
	row, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return userFromRow(row), nil
}

// UpdateUser updates the user's name.
func (s *AuthService) UpdateUser(ctx context.Context, id, name string) error {
	// Fetch current values so we don't overwrite email/password_hash.
	current, err := s.queries.GetUserByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	_, err = s.queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		Name:         name,
		Email:        current.Email,
		PasswordHash: current.PasswordHash,
		UpdatedAt:    timeNow(),
		ID:           id,
	})
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
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
		UserID:      sql.NullString{String: userID, Valid: true},
		Name:        sql.NullString{String: name, Valid: true},
		KeyHash:     sql.NullString{String: keyHash, Valid: true},
		Permissions: "[]",
		CreatedAt:   sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		return "", nil, fmt.Errorf("create api key: %w", err)
	}

	return plainKey, apiKeyFromRow(row), nil
}

// ValidateAPIKey looks up the key by its SHA256 hash, updates last_used_at, and returns the
// owning user ID and key ID.
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (string, string, error) {
	keyHash := hashAPIKey(key)

	row, err := s.queries.GetAPIKeyByHash(ctx, sql.NullString{String: keyHash, Valid: true})
	if err != nil {
		return "", "", errors.New("invalid api key")
	}

	now := timeNow()
	if err := s.queries.UpdateAPIKeyLastUsed(ctx, sqlc.UpdateAPIKeyLastUsedParams{
		LastUsedAt: sql.NullTime{Time: now, Valid: true},
		ID:         row.ID,
	}); err != nil {
		return "", "", fmt.Errorf("update last_used_at: %w", err)
	}

	userID := ""
	if row.UserID.Valid {
		userID = row.UserID.String
	}
	return userID, row.ID, nil
}

// ListAPIKeys returns the API key metadata for the given user (no hashes).
func (s *AuthService) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	rows, err := s.queries.ListAPIKeysByUser(ctx, sql.NullString{String: userID, Valid: true})
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
	rows, err := s.queries.ListAPIKeysByUser(ctx, sql.NullString{String: userID, Valid: true})
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
	rows, err := s.queries.ListAPIKeysByUser(ctx, sql.NullString{String: userID, Valid: true})
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
