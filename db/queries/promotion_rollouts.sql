-- PromotionRollouts are addressed by Project name and rollout name, and by
-- id once one is known. Their Stage, Freight and Targets are names too.

-- Numbers the rollout after the Stage's last one, creating the Stage's row on
-- its first rollout. The Stage's row stays locked until the transaction ends,
-- which serializes the Stage's rollouts; a create that fails rolls the counter
-- back with it, so numbers have no gaps.
-- name: CreatePromotionRollout :one
WITH stage AS (
    INSERT INTO stages (project_name, name, last_promotion_rollout_number)
    VALUES (sqlc.arg(project_name)::text, sqlc.arg(stage_name)::text, 1)
    ON CONFLICT (project_name, name) DO UPDATE SET
        last_promotion_rollout_number = stages.last_promotion_rollout_number + 1
    RETURNING last_promotion_rollout_number
)
INSERT INTO promotion_rollouts (
    project_name, name, stage_name, number, freight_name, update_strategy, created_by
)
SELECT
    sqlc.arg(project_name)::text,
    sqlc.arg(name)::text,
    sqlc.arg(stage_name)::text,
    stage.last_promotion_rollout_number,
    sqlc.arg(freight_name)::text,
    sqlc.arg(update_strategy)::jsonb,
    sqlc.arg(created_by)::text
FROM stage
RETURNING *;

-- name: GetPromotionRollout :one
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- name: GetPromotionRolloutByID :one
SELECT * FROM promotion_rollouts WHERE id = $1;

-- Locks the rollout's row until the transaction ends, so that what it
-- returns is still true when the caller writes.
-- name: GetPromotionRolloutByIDForUpdate :one
SELECT * FROM promotion_rollouts WHERE id = $1 FOR UPDATE;

-- Numbers are per Stage, so a Project's rollouts are ordered by creation time.
-- name: ListPromotionRollouts :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
ORDER BY created_at, stage_name, number;

-- name: ListPromotionRolloutsByStage :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name) AND stage_name = sqlc.arg(stage_name)
ORDER BY number;

-- The rollouts that fan out to the named Target, newest first.
-- name: ListPromotionRolloutsByTarget :many
SELECT promotion_rollouts.* FROM promotion_rollouts
JOIN promotion_rollout_targets
    ON promotion_rollout_targets.promotion_rollout_id = promotion_rollouts.id
WHERE promotion_rollouts.project_name = sqlc.arg(project_name)
    AND promotion_rollout_targets.target_name = sqlc.arg(target_name)
ORDER BY promotion_rollouts.created_at DESC, promotion_rollouts.number DESC;

-- The ids of the open rollouts, paged by id, for a reconciler's resync.
-- name: ListOpenPromotionRolloutIDs :many
SELECT id FROM promotion_rollouts
WHERE phase IN ('Pending', 'Running') AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(row_limit);

-- updated_at doubles as the row's version, so it must grow with every write
-- to a row even across a clock step or two writes in one microsecond.
-- name: UpdatePromotionRolloutStatus :one
UPDATE promotion_rollouts SET
    phase = sqlc.arg(phase)::text,
    message = sqlc.arg(message)::text,
    started_at = sqlc.narg(started_at)::timestamptz,
    finished_at = sqlc.narg(finished_at)::timestamptz,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE id = $1
RETURNING *;

-- name: DeletePromotionRollout :execrows
DELETE FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- Adds a Target to the rollout's fan-out. A Target already present is left
-- alone and nothing is returned.
-- name: AddPromotionRolloutTarget :one
INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_name)
VALUES (sqlc.arg(promotion_rollout_id), sqlc.arg(target_name))
ON CONFLICT DO NOTHING
RETURNING *;

-- name: ListPromotionRolloutTargets :many
SELECT * FROM promotion_rollout_targets
WHERE promotion_rollout_id = ANY(sqlc.arg(promotion_rollout_ids)::uuid[])
ORDER BY promotion_rollout_id, target_name;

-- The Targets a rollout fans out to, as they are now. A Target deleted since
-- the rollout was made is absent; ListPromotionRolloutTargets still has it.
-- name: ListTargetsOfPromotionRollout :many
SELECT targets.* FROM promotion_rollout_targets
JOIN promotion_rollouts
    ON promotion_rollouts.id = promotion_rollout_targets.promotion_rollout_id
JOIN targets
    ON targets.project_name = promotion_rollouts.project_name
    AND targets.name = promotion_rollout_targets.target_name
WHERE promotion_rollout_targets.promotion_rollout_id = $1
ORDER BY targets.name;

-- name: UpdatePromotionRolloutTarget :execrows
UPDATE promotion_rollout_targets SET
    promotion = sqlc.arg(promotion)::text,
    phase = sqlc.arg(phase)::text
WHERE promotion_rollout_id = sqlc.arg(promotion_rollout_id)
    AND target_name = sqlc.arg(target_name);
