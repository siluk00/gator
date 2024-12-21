-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows(id, created_at, updated_at, user_id, feed_id) VALUES(
        $1,
        $2,
        $2,
        $3,
        $4  
    ) RETURNING *
)
SELECT 
    inserted_feed_follow.*,
    users.name AS user_name,
    feeds.name AS feed_name
FROM inserted_feed_follow
    INNER JOIN users ON users.id = inserted_feed_follow.user_id
    INNER JOIN feeds ON feeds.id = inserted_feed_follow.feed_id;

-- name: GetFeedFollowsForUser :many
SELECT feeds.name AS feed_name, users.name AS username FROM feed_follows
INNER JOIN users ON users.id = feed_follows.user_id
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE feed_follows.user_id = (SELECT id FROM users WHERE users.name=$1)
;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows
WHERE user_id = (SELECT id FROM users WHERE users.name = $1) AND
    feed_id = (SELECT id FROM feeds WHERE feeds.url = $2);
