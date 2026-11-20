-- A Project is its namespace, and deleting it deletes everything in it. A
-- Project deleted and recreated under the same name arrives with a new UID
-- while its old row may still hold the name. Run this before UpsertProject
-- to delete that row, so every row that references the old Project cascades
-- away with it instead of carrying over to the new one.
-- name: DeleteReplacedProject :exec
DELETE FROM projects WHERE name = $1 AND id <> $2;

-- name: UpsertProject :exec
INSERT INTO projects (id, name, created_at)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: ListProjects :many
SELECT id, name, created_at, synced_at FROM projects ORDER BY name;

-- name: DeleteProjectByName :exec
DELETE FROM projects WHERE name = $1;
