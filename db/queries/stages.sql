-- name: DeleteReplacedStage :exec
DELETE FROM stages
WHERE project_id = sqlc.arg(project_id)
    AND name = sqlc.arg(name) AND id <> sqlc.arg(id);

-- name: UpsertStage :exec
INSERT INTO stages (id, project_id, name, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    project_id = EXCLUDED.project_id,
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP;

-- name: DeleteStageByName :exec
DELETE FROM stages USING projects
WHERE stages.project_id = projects.id
    AND projects.name = sqlc.arg(project_name)
    AND stages.name = sqlc.arg(name);

-- name: ListStages :many
SELECT id, project_id, name, created_at, synced_at FROM stages ORDER BY id;

-- name: DeleteStagesByID :exec
DELETE FROM stages WHERE id = ANY($1::text[]);
