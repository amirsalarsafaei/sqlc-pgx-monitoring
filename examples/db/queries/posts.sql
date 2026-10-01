-- name: CreatePost :one
INSERT INTO posts (user_id, title, content)
VALUES ($1, $2, $3)
RETURNING id, user_id, title, content, created_at, updated_at;

-- name: CreateComment :one
INSERT INTO comments (post_id, user_id, content)
VALUES ($1, $2, $3)
RETURNING id, post_id, user_id, content, created_at;

-- name: GetPostWithStats :one
SELECT
    p.id,
    p.user_id,
    p.title,
    p.content,
    p.created_at,
    p.updated_at,
    u.username,
    COUNT(c.id) AS comment_count
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN comments c ON p.id = c.post_id
WHERE p.id = $1
GROUP BY p.id, u.username;
