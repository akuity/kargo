-- name: DeleteReplacedProject :exec
DELETE FROM projects
WHERE name = sqlc.arg(name) AND id <> sqlc.arg(id);

-- name: UpsertProject :exec
INSERT INTO projects (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: DeleteProjectByName :exec
DELETE FROM projects WHERE name = $1;

-- name: ListProjects :many
SELECT id, name, created_at, synced_at FROM projects ORDER BY id;

-- name: DeleteProjectsByID :exec
DELETE FROM projects WHERE id = ANY($1::text[]);
