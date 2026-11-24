-- The Promotions to a Target are read from the mirror's rows, which are kept
-- past their Promotions' deletion; see promotions.sql. Nothing here writes.

-- A page of the Promotions to the named Target, newest first: those of every
-- Stage, or of the one named, created before the Promotion the previous page
-- ended at, or the newest when none is given. These are the mirror's rows,
-- which outlive their Promotions in Kubernetes, so the history reaches back
-- past what Kubernetes still has, for as long as each Promotion's Stage
-- lives. A Target is named, not referenced, so this is the history of the
-- name.
-- name: ListTargetPromotions :many
SELECT
    promotions.id,
    promotions.name,
    promotions.stage_name,
    promotions.freight_name,
    COALESCE(NULLIF(promotions.phase, ''), 'Pending')::text AS phase,
    promotions.created_at,
    promotions.finished_at,
    COALESCE(promotion_rollouts.number, 0)::bigint AS rollout_number
FROM promotions
LEFT JOIN promotion_rollouts ON promotion_rollouts.id = promotions.promotion_rollout_id
WHERE promotions.project_name = sqlc.arg(project_name)
    AND promotions.target_name = sqlc.arg(target_name)::text
    AND (sqlc.narg(stage_name)::text IS NULL OR promotions.stage_name = sqlc.narg(stage_name))
    AND (promotions.created_at, promotions.id) < (
        COALESCE(sqlc.narg(before_created_at)::timestamptz, 'infinity'::timestamptz),
        COALESCE(sqlc.narg(before_id)::text, '')
    )
ORDER BY promotions.created_at DESC, promotions.id DESC
LIMIT sqlc.arg(page_size);
