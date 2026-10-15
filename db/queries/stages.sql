-- A Stage deleted and recreated under the same name arrives with a new UID
-- while its old row may still hold the name. Run this before UpsertStage to
-- delete that row, so the upsert does not collide with it.
-- name: DeleteReplacedStage :exec
DELETE FROM stages
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND id <> sqlc.arg(id);

-- name: UpsertStage :exec
INSERT INTO stages (id, project_name, name, created_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(project_name),
    sqlc.arg(name),
    sqlc.arg(created_at)
)
ON CONFLICT (id) DO UPDATE SET
    project_name = EXCLUDED.project_name,
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: ListStages :many
SELECT * FROM stages ORDER BY project_name, name;

-- name: DeleteStageByName :exec
DELETE FROM stages
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);
