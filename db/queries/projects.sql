-- Sync is keyed on the Kubernetes UID, so a row carrying the same name as
-- an incoming Project but a different id can only be left over from a
-- Project that was deleted and recreated. Remove it before upserting so the
-- unique index on name is not violated.
-- name: DeleteReplacedProject :exec
DELETE FROM projects WHERE name = $1 AND id <> $2;

-- name: UpsertProject :exec
INSERT INTO projects (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: GetProjectByName :one
SELECT id, name, created_at, synced_at FROM projects WHERE name = $1;

-- name: ListProjects :many
SELECT id, name, created_at, synced_at FROM projects ORDER BY name;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: DeleteProjectByName :exec
DELETE FROM projects WHERE name = $1;

-- name: DeleteProjectsByID :exec
DELETE FROM projects WHERE id = ANY($1::text[]);
