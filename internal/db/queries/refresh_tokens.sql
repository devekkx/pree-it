-- name: CreateRefreshToken :one
-- Called by: AuthService.issuePairTx (login and refresh paths)
-- used defaults to FALSE via the table DEFAULT — not passed explicitly.
INSERT INTO auth_schema.refresh_tokens (
    id,
    user_id,
    token_hash,
    family,
    expires_at
) VALUES (
    @id,
    @user_id,
    @token_hash,
    @family,
    @expires_at
)
RETURNING
    id,
    user_id,
    token_hash,
    family,
    used,
    expires_at,
    created_at;


-- name: GetRefreshTokenByHash :one
-- Called by: AuthService.Refresh, AuthService.Logout
-- The token_hash column has a UNIQUE index — this is always an index scan.
SELECT
    id,
    user_id,
    token_hash,
    family,
    used,
    expires_at,
    created_at
FROM auth_schema.refresh_tokens
WHERE token_hash = @token_hash
LIMIT 1;


-- name: GetRefreshTokenByID :one
-- Reserved for internal admin/audit use.
SELECT
    id,
    user_id,
    token_hash,
    family,
    used,
    expires_at,
    created_at
FROM auth_schema.refresh_tokens
WHERE id = @id
LIMIT 1;


-- name: MarkRefreshTokenUsed :one
-- Called by: AuthService.Refresh inside ExecTx
-- Marks the consumed token so any replay is detected on the next attempt.
-- Returns the full row so callers can assert the state transition.
UPDATE auth_schema.refresh_tokens
SET used = TRUE
WHERE id = @id
RETURNING
    id,
    user_id,
    token_hash,
    family,
    used,
    expires_at,
    created_at;


-- name: InvalidateTokenFamily :exec
-- Called by: AuthService.Refresh (reuse detected), AuthService.Logout
-- Deletes ALL tokens in the family — burns every session in the login chain.
DELETE FROM auth_schema.refresh_tokens
WHERE family = @family;


-- name: DeleteExpiredTokens :exec
-- Called by: background cleanup goroutine in main.go (every hour).
-- The partial index on (expires_at) WHERE used = FALSE keeps this fast.
DELETE FROM auth_schema.refresh_tokens
WHERE expires_at < NOW();


-- name: DeleteUserTokens :exec
-- Called by: account deletion handler (Phase 2).
-- The FK CASCADE already handles this when DeleteUser fires, but this
-- query is useful when revoking all sessions without deleting the account.
DELETE FROM auth_schema.refresh_tokens
WHERE user_id = @user_id;


-- name: GetActiveTokensByUser :many
-- Called by: session listing handler (Phase 2).
-- Returns non-expired, unused tokens ordered newest first.
SELECT
    id,
    user_id,
    token_hash,
    family,
    used,
    expires_at,
    created_at
FROM auth_schema.refresh_tokens
WHERE user_id    = @user_id
  AND used       = FALSE
  AND expires_at > NOW()
ORDER BY created_at DESC;