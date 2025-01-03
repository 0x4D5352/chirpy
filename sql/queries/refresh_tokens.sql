-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
	$1,
	NOW(),
	NOW(),
	$2,
	CURRENT_DATE + 60
)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: GetRefreshTokens :many
SELECT * FROM refresh_tokens
ORDER BY created_at ASC;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1;

-- name: ResetTokens :exec
DELETE FROM refresh_tokens;
