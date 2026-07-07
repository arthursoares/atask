-- name: CreateInvite :one
INSERT INTO invites (id, email, role, token, created_by, created_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetInviteByToken :one
-- expires_at > datetime('now') is a defense-in-depth clause only; the
-- authoritative expiry check is done in Go (see AuthService.ValidateInviteToken).
SELECT * FROM invites
WHERE token = ? AND claimed_at IS NULL AND expires_at > datetime('now');

-- name: ClaimInvite :execrows
-- claimed_at IS NULL makes this a single-use compare-and-swap: a second,
-- concurrent claim affects 0 rows (see AuthService.ClaimInvite).
UPDATE invites SET claimed_at = ? WHERE id = ? AND claimed_at IS NULL;

-- name: ListInvites :many
SELECT * FROM invites ORDER BY created_at DESC;
