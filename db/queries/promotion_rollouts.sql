-- PromotionRollouts are addressed by Project name and rollout name
-- everywhere, in the database as outside it.

-- Numbers the rollout within its Stage and inserts it in one statement. The
-- number comes from the Stage row's counter, which only this query writes.
-- The UPDATE locks the row, serializing a Stage's creates, and a failure
-- anywhere in the statement rolls the increment back, so numbers are
-- gapless. A Stage deleted and recreated gets a fresh row with the counter
-- at 0, and its predecessor's rollouts go with the old row, so the new
-- Stage's rollouts number from 1. The same locked row supplies the Stage's
-- id, which the rollout references so that it is deleted with the Stage. A
-- Stage the mirror has not recorded has no row, so nothing is inserted. The
-- targets CTE records the rollout's Targets in the same statement (an INSERT
-- in a data-modifying CTE always runs, even though the outer SELECT reads
-- only from rollout), so a Target that no longer exists fails the whole
-- create.
-- name: CreatePromotionRollout :one
WITH stage AS (
    UPDATE stages SET last_promotion_rollout_number = last_promotion_rollout_number + 1
    WHERE project_name = sqlc.arg(project_name)::text
        AND name = sqlc.arg(stage_name)::text
    RETURNING id AS stage_id, last_promotion_rollout_number AS number
), rollout AS (
    INSERT INTO promotion_rollouts (
        project_name,
        name,
        stage_id,
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
        stage.stage_id,
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

-- A Stage's rollout by its number, served by the Stage/number unique key.
-- name: GetPromotionRolloutByNumber :one
SELECT * FROM promotion_rollouts
WHERE project_name = sqlc.arg(project_name)
    AND stage_name = sqlc.arg(stage_name)
    AND number = sqlc.arg(number);

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
-- that the index serves them whether or not a bound is given. A Stage's
-- rollouts are keyed by its row's id, which each list of them finds by the
-- Stage's name first; a Stage the mirror has not recorded has none.

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
WHERE stage_id = (
    SELECT id FROM stages
    WHERE stages.project_name = sqlc.arg(project_name)
        AND stages.name = sqlc.arg(stage_name)
)
    AND number < COALESCE(sqlc.narg(before_number)::bigint, 9223372036854775807)
ORDER BY number DESC
LIMIT sqlc.arg(page_size);

-- A page of a Stage's rollouts of one piece of Freight, newest first, as
-- ListStagePromotionRollouts.
-- name: ListStagePromotionRolloutsOfFreight :many
SELECT * FROM promotion_rollouts
WHERE stage_id = (
    SELECT id FROM stages
    WHERE stages.project_name = sqlc.arg(project_name)
        AND stages.name = sqlc.arg(stage_name)
)
    AND freight_name = sqlc.arg(freight_name)
    AND number < COALESCE(sqlc.narg(before_number)::bigint, 9223372036854775807)
ORDER BY number DESC
LIMIT sqlc.arg(page_size);

-- A page of a Stage's rollouts numbered above after_number, oldest first.
-- name: ListStagePromotionRolloutsAfter :many
SELECT * FROM promotion_rollouts
WHERE stage_id = (
    SELECT id FROM stages
    WHERE stages.project_name = sqlc.arg(project_name)
        AND stages.name = sqlc.arg(stage_name)
)
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
        WHERE promotion_rollouts.stage_id = stages.id
            AND promotion_rollouts.status_phase IN ('Pending', 'Running')
        ORDER BY promotion_rollouts.status_phase = 'Running' DESC, promotion_rollouts.number
        LIMIT 1
    )
    UNION ALL
    (
        SELECT * FROM promotion_rollouts
        WHERE promotion_rollouts.stage_id = stages.id
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

-- What has become of each Target of each of the rollouts: the newest
-- Promotion to it and that Promotion's phase, in rollout then Target name
-- order. One query serves a whole page of rollouts. Among one Target's
-- Promotions, name order is creation order (names embed a ULID), so the
-- newest is the greatest name. A Promotion the controller has not picked
-- up yet has no phase and is reported as Pending. Only Promotions that
-- name a Target count; a rollout's Promotions all do, but the mirror does
-- not enforce it.
--
-- These are the mirror's rows, so they lag the cluster by the mirror's
-- sync. From rough testing that lag is fine for what reads this: the API
-- and Project stats, which present progress rather than act on it.
-- name: ListPromotionRolloutTargetStatuses :many
SELECT DISTINCT ON (promotion_rollout_id, target_name)
    promotion_rollout_id::uuid AS promotion_rollout_id,
    target_name::text AS target_name,
    name AS promotion_name,
    COALESCE(NULLIF(phase, ''), 'Pending')::text AS phase
FROM promotions
WHERE promotion_rollout_id = ANY(sqlc.arg(ids)::uuid[])
    AND target_name IS NOT NULL
ORDER BY promotion_rollout_id, target_name, name DESC;

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

-- Starts a rollout: moves it from Pending to Running and records its
-- FreightCollection in promotion_rollout_freight. Both happen in one
-- statement, so a rollout never starts without its collection.
--
-- The rollout must be Pending and must promote the Freight given. id and
-- updated_at are preconditions as in UpdatePromotionRolloutStatus. If any of
-- these fails, nothing is written and nothing is returned.
--
-- The collection gets one row per origin:
--   * the Freight being promoted, from the arguments;
--   * for each other requested origin, a copy of the previous rollout's row.
--
-- The requested origins are the ones the Stage requests Freight from now.
-- The database does not know them, so the caller passes them as two arrays
-- of equal length, kinds and names.
--
-- The previous rollout is the Stage's newest lower-numbered rollout that has
-- collection rows, whatever phase it ended in. Rollouts without rows are
-- skipped: they were aborted before starting, or started before this table
-- existed, and stopping at one would drop every carried origin.
--
-- A rollout that already has collection rows keeps them unchanged.
-- name: StartPromotionRollout :one
WITH rollout AS (
    UPDATE promotion_rollouts SET
        status_phase = 'Running',
        status_reason = sqlc.arg(status_reason)::text,
        status_message = sqlc.arg(status_message)::text,
        status_started_at = sqlc.narg(status_started_at)::timestamptz,
        status_finished_at = sqlc.narg(status_finished_at)::timestamptz,
        updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
    WHERE promotion_rollouts.project_name = sqlc.arg(project_name)
        AND promotion_rollouts.name = sqlc.arg(name)
        AND (sqlc.narg(id)::uuid IS NULL OR promotion_rollouts.id = sqlc.narg(id))
        AND (
            sqlc.narg(updated_at)::timestamptz IS NULL
            OR promotion_rollouts.updated_at = sqlc.narg(updated_at)
        )
        AND promotion_rollouts.status_phase = 'Pending'
        AND promotion_rollouts.freight_name = sqlc.arg(freight_name)::text
    RETURNING *
), previous AS (
    SELECT earlier.id
    FROM rollout
    JOIN promotion_rollouts AS earlier ON earlier.stage_id = rollout.stage_id
    WHERE earlier.number < rollout.number
        AND EXISTS (
            SELECT 1 FROM promotion_rollout_freight AS recorded
            WHERE recorded.promotion_rollout_id = earlier.id
        )
    ORDER BY earlier.number DESC
    LIMIT 1
), freight AS (
    INSERT INTO promotion_rollout_freight (
        promotion_rollout_id,
        origin_kind,
        origin_name,
        freight_name,
        reference
    )
    SELECT
        rollout.id,
        collected.origin_kind,
        collected.origin_name,
        collected.freight_name,
        collected.reference
    FROM rollout
    CROSS JOIN LATERAL (
        SELECT
            sqlc.arg(origin_kind)::text AS origin_kind,
            sqlc.arg(origin_name)::text AS origin_name,
            sqlc.arg(freight_name)::text AS freight_name,
            sqlc.arg(reference)::json AS reference
        UNION ALL
        SELECT
            carried.origin_kind,
            carried.origin_name,
            carried.freight_name,
            carried.reference
        FROM promotion_rollout_freight AS carried
        WHERE carried.promotion_rollout_id = (SELECT previous.id FROM previous)
            -- Set-returning functions in one select list advance together,
            -- which zips the arrays.
            AND (carried.origin_kind, carried.origin_name) IN (
                SELECT
                    unnest(sqlc.arg(requested_origin_kinds)::text[]),
                    unnest(sqlc.arg(requested_origin_names)::text[])
            )
            AND (carried.origin_kind, carried.origin_name)
                <> (sqlc.arg(origin_kind)::text, sqlc.arg(origin_name)::text)
    ) AS collected
    WHERE NOT EXISTS (
        SELECT 1 FROM promotion_rollout_freight AS existing
        WHERE existing.promotion_rollout_id = rollout.id
    )
)
SELECT * FROM rollout;

-- A rollout's FreightCollection, one row per origin, in origin order. A
-- rollout started before collections were recorded has no rows.
-- name: ListPromotionRolloutFreight :many
SELECT * FROM promotion_rollout_freight
WHERE promotion_rollout_id = sqlc.arg(promotion_rollout_id)
ORDER BY origin_kind, origin_name;

-- Records a request to abort one of a Stage's rollouts, by its number, and
-- returns the row as it now stands. Only a rollout that has not finished can
-- be aborted; one that has is left alone and nothing is returned. The first
-- request is kept: a second leaves who asked and when as they were.
--
-- A Pending rollout has created no Promotions, so it ends here, as Aborted
-- with the given reason and message, never having started. A Running one
-- keeps its status, which the reconciler owns: it acts on the request. Every
-- SET expression reads the row as it was before the update.
-- name: AbortPromotionRollout :one
UPDATE promotion_rollouts SET
    abort_requested_at = COALESCE(abort_requested_at, clock_timestamp()),
    abort_requested_by = CASE
        WHEN abort_requested_at IS NULL THEN sqlc.arg(requested_by)::text
        ELSE abort_requested_by
    END,
    status_phase = CASE
        WHEN status_phase = 'Pending' THEN 'Aborted'
        ELSE status_phase
    END,
    status_reason = CASE
        WHEN status_phase = 'Pending' THEN sqlc.arg(status_reason)::text
        ELSE status_reason
    END,
    status_message = CASE
        WHEN status_phase = 'Pending' THEN sqlc.arg(status_message)::text
        ELSE status_message
    END,
    status_finished_at = CASE
        WHEN status_phase = 'Pending' THEN clock_timestamp()
        ELSE status_finished_at
    END,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND stage_name = sqlc.arg(stage_name)
    AND number = sqlc.arg(number)
    AND status_phase IN ('Pending', 'Running')
RETURNING *;
