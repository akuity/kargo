-- PromotionRollouts are addressed by Project name and rollout name
-- everywhere, in the database as outside it.

-- Numbers the rollout within its Stage and inserts it in one statement, so
-- that both land or neither does without a transaction. The upsert on the
-- Stage's row locks it, which serializes a Stage's creates; a failure
-- anywhere in the statement rolls the increment back, so numbers stay
-- gapless. The rollout's Targets are resolved later; see
-- SetPromotionRolloutTargets.
-- name: CreatePromotionRollout :one
WITH stage AS (
    INSERT INTO stages (project_name, name, last_promotion_rollout_number)
    VALUES (sqlc.arg(project_name)::text, sqlc.arg(stage_name)::text, 1)
    ON CONFLICT (project_name, name) DO UPDATE
        SET last_promotion_rollout_number = stages.last_promotion_rollout_number + 1
    RETURNING last_promotion_rollout_number AS number
), rollout AS (
    INSERT INTO promotion_rollouts (
        project_name, name, stage_name, number, freight_name, max_concurrent, created_by
    )
    SELECT
        sqlc.arg(project_name)::text,
        sqlc.arg(name)::text,
        sqlc.arg(stage_name)::text,
        stage.number,
        sqlc.arg(freight_name)::text,
        sqlc.arg(max_concurrent)::bigint,
        sqlc.arg(created_by)::text
    FROM stage
    RETURNING *
)
SELECT * FROM rollout;

-- Records the Targets a rollout promotes to and marks it resolved, in one
-- statement. Only a rollout not yet resolved is written; one that is yields
-- no row and is left alone, so the snapshot is taken exactly once however
-- many reconcilers try. updated_at advances as for any write.
-- name: SetPromotionRolloutTargets :one
WITH resolved AS (
    UPDATE promotion_rollouts SET
        targets_resolved_at = CURRENT_TIMESTAMP,
        updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
    WHERE id = sqlc.arg(id) AND targets_resolved_at IS NULL
    RETURNING *
), targets AS (
    INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_id)
    SELECT resolved.id, unnest(sqlc.arg(target_ids)::uuid[]) FROM resolved
)
SELECT * FROM resolved;

-- name: GetPromotionRollout :one
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- The named rollouts of a Project, for a reader that already knows which
-- ones it wants. Names that do not exist are simply absent from the result.
-- name: ListPromotionRolloutsByName :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND name = ANY(sqlc.arg(names)::text[])
ORDER BY name;

-- A Project's rollouts, or one Stage's when stage_name is given, in
-- creation order.
-- name: ListPromotionRollouts :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND (sqlc.narg(stage_name)::text IS NULL OR stage_name = sqlc.narg(stage_name))
ORDER BY created_at, number;

-- The keys of every rollout not yet in a terminal phase, for the
-- reconciler's boot relist and resync. The predicate matches the partial
-- index promotion_rollouts_open_idx.
-- name: ListOpenPromotionRollouts :many
SELECT project_name, name FROM promotion_rollouts
WHERE status_phase IN ('Pending', 'Running')
ORDER BY project_name, name;

-- The Targets of the given rollouts, with each Target's name, in Target
-- name order within a rollout.
-- name: ListPromotionRolloutTargets :many
SELECT t.promotion_rollout_id, t.target_id, targets.name AS target_name
FROM promotion_rollout_targets t
JOIN targets ON targets.id = t.target_id
WHERE t.promotion_rollout_id = ANY(sqlc.arg(ids)::uuid[])
ORDER BY t.promotion_rollout_id, targets.name;

-- updated_at doubles as the row's version, so it must grow with every write
-- to a row even across a clock step or two writes in one microsecond. A
-- non-null id or updated_at is a precondition; a row that fails one is left
-- alone and nothing is returned.
-- name: UpdatePromotionRolloutStatus :one
UPDATE promotion_rollouts SET
    status_phase = sqlc.arg(status_phase)::text,
    status_reason = sqlc.arg(status_reason)::text,
    status_message = sqlc.arg(status_message)::text,
    status_started_at = sqlc.narg(status_started_at)::timestamptz,
    status_finished_at = sqlc.narg(status_finished_at)::timestamptz,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND (sqlc.narg(id)::uuid IS NULL OR id = sqlc.narg(id))
    AND (
        sqlc.narg(updated_at)::timestamptz IS NULL
        OR updated_at = sqlc.narg(updated_at)
    )
RETURNING *;
