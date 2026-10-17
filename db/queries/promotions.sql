-- Promotion rows are only ever inserted and updated; see the migration that
-- creates the table. There is no delete.

-- name: UpsertPromotion :exec
INSERT INTO promotions (
    id, project_name, name, stage_name, freight_name, target_name, phase, created_at, finished_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(project_name),
    sqlc.arg(name),
    sqlc.arg(stage_name),
    sqlc.arg(freight_name),
    sqlc.narg(target_name),
    sqlc.arg(phase),
    sqlc.arg(created_at),
    sqlc.narg(finished_at)
)
ON CONFLICT (id) DO UPDATE SET
    project_name = EXCLUDED.project_name,
    name = EXCLUDED.name,
    stage_name = EXCLUDED.stage_name,
    freight_name = EXCLUDED.freight_name,
    target_name = EXCLUDED.target_name,
    phase = EXCLUDED.phase,
    created_at = EXCLUDED.created_at,
    finished_at = EXCLUDED.finished_at,
    synced_at = CURRENT_TIMESTAMP;

-- The table grows without bound, so the mirror never lists all of it: it
-- asks only after the rows of the Promotions Kubernetes currently has.
-- name: ListPromotionsByID :many
SELECT * FROM promotions
WHERE id = ANY(sqlc.arg(ids)::text[])
ORDER BY id;
