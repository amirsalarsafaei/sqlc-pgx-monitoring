-- User queries
-- name: CreateUser :one
INSERT INTO users (username, email)
VALUES ($1, $2)
RETURNING id, username, email, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, email, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
SELECT id, username, email, created_at, updated_at
FROM users
WHERE username = $1;

-- name: ListUsers :many
SELECT id, username, email, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUserEmail :one
UPDATE users
SET email = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, username, email, created_at, updated_at;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- Post queries
-- name: CreatePost :one
INSERT INTO posts (user_id, title, content)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, content, created_at, updated_at;

-- name: GetPostByID :one
SELECT id, user_id, title, content, created_at, updated_at
FROM posts
WHERE id = $1;

-- name: ListPostsByUserID :many
SELECT id, user_id, title, content, created_at, updated_at
FROM posts
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAllPosts :many
SELECT id, user_id, title, content, created_at, updated_at
FROM posts
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeletePost :exec
DELETE FROM posts WHERE id = $1;

-- Comment queries
-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id, post_id, user_id, content, created_at;

-- name: ListCommentsByPostID :many
SELECT id, post_id, user_id, content, created_at
FROM comments
WHERE post_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;

-- Complex query - post with user and comment count
-- name: GetPostWithStats :one
SELECT 
    p.id, 
    p.user_id, 
    p.title, 
    p.content, 
    p.created_at, 
    p.updated_at,
    u.username,
    COUNT(c.id) as comment_count
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
WHERE p.id = $1
GROUP BY p.id, u.username;

-- Batch query - sqlc batches every queued statement under the same query name
-- name: InsertUsers :batchexec
INSERT INTO users (username, email)
VALUES ($1, $2);
