-- name: InsertDeltaEvent :exec
INSERT INTO delta_events (
    entity_type, entity_id, action, field, old_value, new_value, actor_id, timestamp, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: ListDeltaEventsSince :many
SELECT * FROM delta_events
WHERE id > ? AND user_id = ?
ORDER BY id;

-- name: InsertDomainEvent :one
INSERT INTO domain_events (
    type, entity_type, entity_id, actor_id, payload, timestamp, user_id
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
)
RETURNING id;

-- name: ListDomainEventsSince :many
SELECT * FROM domain_events
WHERE id > ? AND user_id = ?
ORDER BY id;

-- name: ListDomainEventsByTypeSince :many
SELECT * FROM domain_events
WHERE type = ? AND id > ? AND user_id = ?
ORDER BY id;

-- name: ListDomainEventsByEntitySince :many
SELECT * FROM domain_events
WHERE entity_type = ? AND entity_id = ? AND id > ? AND user_id = ?
ORDER BY id;
