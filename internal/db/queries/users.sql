-- name: CreateUser :one
-- Called by: AuthService.Register
INSERT INTO auth_schema.users (
    id,
    email,
    password_hash,
    is_verified
) VALUES (
    @id,
    @email,
    @password_hash,
    @is_verified
)
RETURNING
    id,
    email,
    password_hash,
    is_verified,
    created_at,
    updated_at;


-- name: GetUserByEmail :one
-- Called by: AuthService.Login, AuthService.Register (existence check)
SELECT
    id,
    email,
    password_hash,
    is_verified,
    created_at,
    updated_at
FROM auth_schema.users
WHERE email = @email
LIMIT 1;


-- name: GetUserByID :one
-- Called by: AuthService.Refresh (resolve user after token lookup)
SELECT
    id,
    email,
    password_hash,
    is_verified,
    created_at,
    updated_at
FROM auth_schema.users
WHERE id = @id
LIMIT 1;


-- name: UpdateUserPassword :one
-- Called by: AuthService.Login transparent rehash (NeedsRehash path)
UPDATE auth_schema.users
SET
    password_hash = @password_hash,
    updated_at    = NOW()
WHERE id = @id
RETURNING
    id,
    email,
    password_hash,
    is_verified,
    created_at,
    updated_at;


-- name: MarkUserVerified :one
-- Called by: email verification handler (Phase 2)
UPDATE auth_schema.users
SET
    is_verified = TRUE,
    updated_at  = NOW()
WHERE id = @id
RETURNING
    id,
    email,
    password_hash,
    is_verified,
    created_at,
    updated_at;


-- name: DeleteUser :exec
-- Called by: account deletion handler (Phase 2)
-- Cascade in the FK deletes all refresh_tokens rows automatically.
DELETE FROM auth_schema.users
WHERE id = @id;