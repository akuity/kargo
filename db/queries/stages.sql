-- A Stage deleted and recreated under the same name arrives with a new UID
-- while its old row may still hold the name. Run this before UpsertStage to
-- delete that row, so the upsert does not collide with it.
-- name: DeleteReplacedStage :exec
DELETE FROM stages
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND id <> sqlc.arg(id);

-- The rollout counter is not the mirror's to write: an insert starts it at
-- zero and an update leaves it alone. An update that would change nothing
-- the mirror writes is skipped, synced_at included.
-- name: UpsertStage :exec
INSERT INTO stages (id, project_name, name, target_selectors, created_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(project_name),
    sqlc.arg(name),
    sqlc.narg(target_selectors)::jsonb,
    sqlc.arg(created_at)
)
ON CONFLICT (id) DO UPDATE SET
    project_name = EXCLUDED.project_name,
    name = EXCLUDED.name,
    target_selectors = EXCLUDED.target_selectors,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP
WHERE (
    stages.project_name,
    stages.name,
    stages.target_selectors,
    stages.created_at
) IS DISTINCT FROM (
    EXCLUDED.project_name,
    EXCLUDED.name,
    EXCLUDED.target_selectors,
    EXCLUDED.created_at
);

-- name: ListStages :many
SELECT * FROM stages ORDER BY project_name, name;

-- name: DeleteStageByName :exec
DELETE FROM stages
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);
