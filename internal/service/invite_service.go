package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	sqlc "github.com/atask/atask/internal/store/sqlc"
	"github.com/google/uuid"
)

// inviteTTL is the default lifetime of a freshly created invite token.
const inviteTTL = 7 * 24 * time.Hour

// Invite is the domain type for an invite token. Token is only meaningful on
// the object CreateInvite returns (the caller embeds it in the invite URL);
// ListInvites callers should decide for themselves whether it's safe to
// expose a claimed/expired invite's token in a response.
type Invite struct {
	ID        string
	Email     string
	Role      string
	Token     string
	CreatedBy string
	CreatedAt time.Time
	ClaimedAt *time.Time
	ExpiresAt time.Time
}

// ErrInviteInvalid is returned by ValidateInviteToken for any invite that
// cannot be claimed right now — unknown token, already claimed, or expired.
// Deliberately a single sentinel (not distinct errors per cause) so callers
// can't build a token-guessing oracle from which error came back.
var ErrInviteInvalid = errors.New("invalid or expired invite")

// ErrInviteAlreadyClaimed is returned by ClaimInvite when its compare-and-swap
// UPDATE affects zero rows — a concurrent request already claimed the same
// invite between the caller's ValidateInviteToken read and this write.
var ErrInviteAlreadyClaimed = errors.New("invite already claimed")

// ErrInvalidInviteRole is returned by CreateInvite when role is anything
// other than "user" or "admin".
var ErrInvalidInviteRole = errors.New(`invite role must be "user" or "admin"`)

func inviteFromRow(row sqlc.Invite) *Invite {
	inv := &Invite{
		ID:        row.ID,
		Email:     row.Email,
		Role:      row.Role,
		Token:     row.Token,
		CreatedBy: row.CreatedBy,
		CreatedAt: row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
	}
	if row.ClaimedAt.Valid {
		t := row.ClaimedAt.Time
		inv.ClaimedAt = &t
	}
	return inv
}

// CreateInvite generates a random token and persists a new invite for
// email+role (role must be "user" or "admin"), expiring inviteTTL from now.
// The returned Invite's Token is populated so the caller (internal/api) can
// build the invite URL ({BaseURL}/invite/{token}).
func (s *AuthService) CreateInvite(ctx context.Context, email, role, createdBy string) (*Invite, error) {
	if role != "user" && role != "admin" {
		return nil, ErrInvalidInviteRole
	}
	if email == "" {
		return nil, errors.New("email is required")
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generate invite token: %w", err)
	}
	token := hex.EncodeToString(raw)

	now := timeNow()
	row, err := s.queries.CreateInvite(ctx, sqlc.CreateInviteParams{
		ID:        uuid.New().String(),
		Email:     email,
		Role:      role,
		Token:     token,
		CreatedBy: createdBy,
		CreatedAt: now,
		ExpiresAt: now.Add(inviteTTL),
	})
	if err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}
	return inviteFromRow(row), nil
}

// ValidateInviteToken looks up an invite by token and confirms it is
// unclaimed and unexpired, returning ErrInviteInvalid otherwise.
//
// invites.sql's GetInviteByToken keeps an `expires_at > datetime('now')`
// clause for documentation/defense-in-depth, but it is NOT relied upon here:
// modernc.org/sqlite serializes a bound time.Time as text that includes a
// zone abbreviation (confirmed empirically — e.g.
// "2026-07-13 14:53:28.801966 +0200 CEST" — which SQLite's datetime()
// function, and therefore this '>' comparison, cannot parse), so that clause
// matches expired rows too. The authoritative expiry check below is against
// ExpiresAt as decoded by database/sql into a real time.Time — the same
// pattern ValidateAPIKey uses for api_keys.expires_at (auth_service.go).
func (s *AuthService) ValidateInviteToken(ctx context.Context, token string) (*Invite, error) {
	if token == "" {
		return nil, ErrInviteInvalid
	}
	row, err := s.queries.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, ErrInviteInvalid
	}
	inv := inviteFromRow(row)
	if inv.ClaimedAt != nil {
		return nil, ErrInviteInvalid
	}
	if !inv.ExpiresAt.After(timeNow()) {
		return nil, ErrInviteInvalid
	}
	return inv, nil
}

// ClaimInvite marks the invite claimed via a single-use compare-and-swap
// UPDATE (claimed_at IS NULL in the WHERE clause, invites.sql). Callers MUST
// claim before creating the account (not after) so that a lost race never
// creates a duplicate account: if a concurrent request already claimed the
// same invite, this returns ErrInviteAlreadyClaimed and the caller must not
// proceed to AuthProvider.CreateUser.
func (s *AuthService) ClaimInvite(ctx context.Context, id string) error {
	now := timeNow()
	n, err := s.queries.ClaimInvite(ctx, sqlc.ClaimInviteParams{
		ClaimedAt: sql.NullTime{Time: now, Valid: true},
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("claim invite: %w", err)
	}
	if n == 0 {
		return ErrInviteAlreadyClaimed
	}
	return nil
}

// ListInvites returns all invites, most recently created first. Exposed for
// future admin tooling; no HTTP route uses it yet (Task 17 only wires
// CreateInvite/ClaimInvite/gated-Register).
func (s *AuthService) ListInvites(ctx context.Context) ([]*Invite, error) {
	rows, err := s.queries.ListInvites(ctx)
	if err != nil {
		return nil, fmt.Errorf("list invites: %w", err)
	}
	invites := make([]*Invite, len(rows))
	for i, row := range rows {
		invites[i] = inviteFromRow(row)
	}
	return invites, nil
}
