-- name: CreateFeedFollow :one
WITH new_feed_follow AS (
    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *
)
SELECT
    new_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM new_feed_follow
INNER JOIN feeds ON new_feed_follow.feed_id = feeds.id
INNER JOIN users ON new_feed_follow.user_id = users.id;

-- name: GetFeedFollowsForUser :many
SELECT feeds.name AS feed_name, users.name AS user_name
FROM feed_follows
INNER JOIN feeds ON feed_follows.feed_id = feeds.id
INNER JOIN users ON feed_follows.user_id = users.id
WHERE users.name = $1;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows WHERE user_id = $1 AND feed_id = $2;