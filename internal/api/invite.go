package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/atask/atask/internal/auth"
	"github.com/atask/atask/internal/service"
)

// CreateInvite handles POST /auth/invites — admin-only (see requireAdminAPI
// in routes.go). Body: {"email", "role"}. Generates a random token, persists
// an invite expiring 7 days from now, and returns the invite plus a ready-to
// -share URL ({BaseURL}/invite/{token}).
func (h *AuthHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	adminID := UserIDFromContext(r.Context())

	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	inv, err := h.authSvc.CreateInvite(r.Context(), body.Email, body.Role, adminID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInviteRole) {
			RespondError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("create invite failed", "err", err)
		RespondError(w, http.StatusUnprocessableEntity, "could not create invite")
		return
	}

	RespondJSON(w, http.StatusCreated, map[string]any{
		"id":        inv.ID,
		"email":     inv.Email,
		"role":      inv.Role,
		"token":     inv.Token,
		"expiresAt": inv.ExpiresAt,
		"url":       h.cfg.BaseURL + "/invite/" + inv.Token,
	})
}

// claimInviteBody is the shared request shape for claiming an invite: the
// dedicated ClaimInvite endpoint and Register's !RegistrationOpen path both
// decode into this (Register's body additionally carries Email, which is
// ignored on the invite path — see Register's doc comment).
type claimInviteBody struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// ClaimInvite handles POST /auth/invites/claim — public. Body:
// {"token", "password", "name"}. Validates the invite, creates the account
// via the AuthProvider using the invite's own email+role (never the
// request's — there is none to smuggle one in from), marks the invite
// claimed, and returns the created user's profile. Delegates to claimInvite,
// the same helper Register's gated path uses, so both endpoints enforce
// identical single-use/expiry semantics without duplicating the logic.
func (h *AuthHandler) ClaimInvite(w http.ResponseWriter, r *http.Request) {
	var body claimInviteBody
	if err := DecodeJSON(r, &body); err != nil {
		RespondDecodeError(w, err)
		return
	}

	user, err := h.claimInvite(r.Context(), body.Token, body.Password, body.Name)
	if err != nil {
		respondClaimInviteError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, userJSON(user))
}

// claimInvite is the single implementation of "redeem an invite token into a
// user account", shared by ClaimInvite and Register's !RegistrationOpen
// path. Ordering matters for single-use correctness: it claims the invite
// (the compare-and-swap UPDATE in AuthService.ClaimInvite) BEFORE creating
// the account, so a lost race (two concurrent claims of the same token)
// fails at the claim step and never creates a duplicate account. The
// account's email and role always come from the invite record itself, never
// from caller input — an invite is authorization to become a specific
// pre-declared identity, not a blank check to self-assign a role.
//
// Known gap: if AuthProvider.CreateUser fails after a successful claim (e.g.
// the invited email is already registered), the invite is left burned with
// no account created. Recovering requires an admin to issue a fresh invite;
// see the Task 17 report for why this tradeoff was chosen over the
// alternative (create-then-claim, which instead risks a double-redemption
// race creating two accounts for one invite).
func (h *AuthHandler) claimInvite(ctx context.Context, token, password, name string) (*auth.User, error) {
	inv, err := h.authSvc.ValidateInviteToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if err := h.authSvc.ClaimInvite(ctx, inv.ID); err != nil {
		return nil, err
	}

	user, err := h.authProvider.CreateUser(inv.Email, password, name, inv.Role)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// respondClaimInviteError maps claimInvite's error cases to HTTP responses.
// Invalid/expired and already-claimed both surface as 403 with the same
// "invite required" framing invite-gated Register uses elsewhere in this
// file — a bad token shouldn't leak whether it was never valid, expired, or
// already used.
func respondClaimInviteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInviteInvalid), errors.Is(err, service.ErrInviteAlreadyClaimed):
		RespondError(w, http.StatusForbidden, "invalid or expired invite")
	default:
		slog.Error("claim invite failed", "err", err)
		RespondError(w, http.StatusUnprocessableEntity, "could not create account")
	}
}
