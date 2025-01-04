-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2
)
RETURNING *;

-- name: FindUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: FindUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUser :exec
UPDATE users
SET email = $1, hashed_password = $2, updated_at = NOW()
WHERE id = $3;

-- name: ResetUsers :exec
DELETE FROM users;
