-- PromotionRollouts are addressed by Project name and rollout name
-- everywhere, in the database as outside it.

-- Numbers the rollout within its Stage and inserts it in one statement, so
-- that both land or neither does without a transaction. The Stage's row is
-- the mirror's, which never writes the counter; the update locks the row,
-- which serializes a Stage's creates, and a failure anywhere in the
-- statement rolls the increment back, so numbers stay gapless. A Stage
-- deleted and recreated under the same name gets a new row whose counter
-- starts over, so the next number is also kept past any the name's earlier
-- rollouts hold. A Stage the mirror has not recorded yet has no row to
-- number from, and the statement inserts nothing. The rollout's Targets are
-- recorded in the same statement, so a Target that no longer exists fails
-- the whole create.
-- name: CreatePromotionRollout :one
WITH stage AS (
    UPDATE stages SET last_promotion_rollout_number = GREATEST(
        stages.last_promotion_rollout_number,
        (
            SELECT COALESCE(MAX(number), 0) FROM promotion_rollouts
            WHERE project_name = sqlc.arg(project_name)::text
                AND stage_name = sqlc.arg(stage_name)::text
        )
    ) + 1
    WHERE project_name = sqlc.arg(project_name)::text
        AND name = sqlc.arg(stage_name)::text
    RETURNING last_promotion_rollout_number AS number
), rollout AS (
    INSERT INTO promotion_rollouts (
        project_name,
        name,
        stage_name,
        number,
        freight_name,
        max_concurrent,
        max_failures,
        created_by
    )
    SELECT
        sqlc.arg(project_name)::text,
        sqlc.arg(name)::text,
        sqlc.arg(stage_name)::text,
        stage.number,
        sqlc.arg(freight_name)::text,
        sqlc.arg(max_concurrent)::bigint,
        sqlc.arg(max_failures)::bigint,
        sqlc.arg(created_by)::text
    FROM stage
    RETURNING *
), targets AS (
    INSERT INTO promotion_rollout_targets (promotion_rollout_id, target_id)
    SELECT rollout.id, unnest(sqlc.arg(target_ids)::uuid[]) FROM rollout
)
SELECT * FROM rollout;

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

-- A Project or a Stage may have had any number of rollouts, so none of the
-- lists below is unbounded: each reads a page of at most page_size rows
-- from an index, and a caller wanting more asks again from where the page
-- ended. Bounds are compared with COALESCE rather than an IS NULL test so
-- that the index serves them whether or not a bound is given.

-- A page of a Project's rollouts, newest first: those created before the
-- given rollout, or the newest when none is given.
-- name: ListProjectPromotionRollouts :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND (created_at, id) < (
        COALESCE(sqlc.narg(before_created_at)::timestamptz, 'infinity'::timestamptz),
        COALESCE(sqlc.narg(before_id)::uuid, 'ffffffff-ffff-ffff-ffff-ffffffffffff'::uuid)
    )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size);

-- A page of a Stage's rollouts, newest first: those numbered below
-- before_number, or the newest when it is NULL.
-- name: ListStagePromotionRollouts :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND stage_name = sqlc.arg(stage_name)
    AND number < COALESCE(sqlc.narg(before_number)::bigint, 9223372036854775807)
ORDER BY number DESC
LIMIT sqlc.arg(page_size);

-- A page of a Stage's rollouts of one piece of Freight, newest first, as
-- ListStagePromotionRollouts.
-- name: ListStagePromotionRolloutsOfFreight :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND stage_name = sqlc.arg(stage_name)
    AND freight_name = sqlc.arg(freight_name)
    AND number < COALESCE(sqlc.narg(before_number)::bigint, 9223372036854775807)
ORDER BY number DESC
LIMIT sqlc.arg(page_size);

-- A page of a Stage's rollouts numbered above after_number, oldest first.
-- name: ListStagePromotionRolloutsAfter :many
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND stage_name = sqlc.arg(stage_name)
    AND number > sqlc.arg(after_number)
ORDER BY number
LIMIT sqlc.arg(page_size);

-- The latest rollouts of each of a Project's Stages, or of the one named:
-- the rollout the Stage is promoting through, if any -- its Running one,
-- else its oldest Pending one -- and the newest that has finished, if any.
-- A Stage has at most one of each. Only Stages the mirror has recorded are
-- read. The open rollout comes from the partial index of open rollouts and
-- the finished one from the newest end of the Stage's numbers, so neither
-- reads more than a few rows however many rollouts a Stage has had.
-- name: ListLatestPromotionRollouts :many
SELECT latest.*
FROM stages
CROSS JOIN LATERAL (
    (
        SELECT * FROM promotion_rollouts
        WHERE promotion_rollouts.project_name = stages.project_name
            AND promotion_rollouts.stage_name = stages.name
            AND promotion_rollouts.status_phase IN ('Pending', 'Running')
        ORDER BY promotion_rollouts.status_phase = 'Running' DESC, promotion_rollouts.number
        LIMIT 1
    )
    UNION ALL
    (
        SELECT * FROM promotion_rollouts
        WHERE promotion_rollouts.project_name = stages.project_name
            AND promotion_rollouts.stage_name = stages.name
            AND promotion_rollouts.status_phase NOT IN ('Pending', 'Running')
        ORDER BY promotion_rollouts.number DESC
        LIMIT 1
    )
) AS latest
WHERE stages.project_name = sqlc.arg(project_name)
    AND (sqlc.narg(stage_name)::text IS NULL OR stages.name = sqlc.narg(stage_name))
ORDER BY latest.stage_name, latest.number;

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

-- Moves the rollout from Pending to Running, if it is the next of its
-- Stage's rollouts to run: the oldest that has not finished, with none
-- Running. Rounds of fan-out are serialized here, in number order. The
-- partial unique index promotion_rollouts_running_key is the final guard:
-- two claims that both pass this statement's checks cannot both commit. No
-- row is returned when the rollout is not Pending or not next; a unique
-- violation says another rollout of the Stage is Running.
-- name: StartPromotionRollout :one
UPDATE promotion_rollouts SET
    status_phase = 'Running',
    status_started_at = COALESCE(status_started_at, CURRENT_TIMESTAMP),
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE promotion_rollouts.id = sqlc.arg(id)
    AND promotion_rollouts.status_phase = 'Pending'
    AND NOT EXISTS (
        SELECT 1 FROM promotion_rollouts ahead
        WHERE ahead.project_name = promotion_rollouts.project_name
            AND ahead.stage_name = promotion_rollouts.stage_name
            AND ahead.status_phase IN ('Pending', 'Running')
            AND ahead.number < promotion_rollouts.number
    )
RETURNING *;

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
