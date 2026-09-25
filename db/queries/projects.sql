-- name: UpsertProject :exec
INSERT INTO projects (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: GetProject :one
SELECT id, name, created_at, synced_at FROM projects WHERE id = $1;

-- name: ListProjects :many
SELECT id, name, created_at, synced_at FROM projects ORDER BY name;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
