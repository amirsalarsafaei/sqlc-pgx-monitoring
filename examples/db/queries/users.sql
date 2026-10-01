-- name: CreateUser :one
INSERT INTO users (username, email)
VALUES ($1, $2)
RETURNING id, username, email, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, created_at, updated_at
FROM users
ORDER BY id DESC
LIMIT $1;

-- name: UpdateUserEmail :one
UPDATE users
SET email = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, username, email, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: InsertUsers :batchexec
INSERT INTO users (username, email)
VALUES ($1, $2);

-- name: GetUsersByID :batchone
SELECT id, username, email, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CopyUsers :copyfrom
INSERT INTO users (username, email)
VALUES ($1, $2);
