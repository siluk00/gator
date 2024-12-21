-- name: CreateFeed :one
INSERT INTO feeds(id, name, url, user_id) VALUES(
    $1,
    $2,
    $3, 
    $4
)
RETURNING *;

-- name: GetFeed :many
SELECT * FROM feeds;

-- name: GetFeedsByUrl :one
SELECT * FROM feeds
    WHERE feeds.url = $1;

-- name: DeleteFeeds :exec
DELETE FROM feeds;

-- name: MarkfeedFetched :exec
UPDATE feeds
SET updated_at = NOW(), last_fetched_at = NOW()
WHERE feeds.id = $1;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;