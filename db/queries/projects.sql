-- Sync is keyed on the Kubernetes UID, but a Project deleted and recreated
-- under the same name arrives with a new UID while its old row still holds
-- the name. Conflicting on name lets one statement replace that row's
-- identity in place. Names and UIDs are immutable in Kubernetes, so a live
-- Project can never collide with another live row on either column.
-- name: UpsertProject :exec
INSERT INTO projects (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (name) DO UPDATE SET
    id = EXCLUDED.id,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: ListProjects :many
SELECT id, name, created_at, synced_at FROM projects ORDER BY name;

-- name: DeleteProjectByName :exec
DELETE FROM projects WHERE name = $1;
