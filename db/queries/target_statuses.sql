-- Where a Target stands with each Stage that governs it is not recorded
-- anywhere: it is read, on request, from the rows that already exist. The
-- mirrored Promotions say what was promoted to a Target and how it went,
-- their promotion_freight rows what a successful one left there, and the
-- PromotionRollouts and their Target snapshots which Targets a rollout has
-- yet to reach or never did. Nothing here writes, and nothing here decides
-- which Stage governs which Target: the caller works that out from the
-- Stages' selectors and passes the pairs in.
--
-- These are the mirror's rows, so what is read lags the cluster by the
-- mirror's sync, and a Promotion names its Target rather than referencing
-- its row, so a Target deleted and recreated under the same name takes up
-- the history of the name.

-- What governance is worked out from: every Target of the Project with its
-- labels and nothing else, in byte order of name, which is the order the
-- caller pages in.
-- name: ListTargetLabels :many
SELECT id, name, labels FROM targets
WHERE project_name = sqlc.arg(project_name)
ORDER BY name COLLATE "C";

-- The status of each of the given pairs of a Target and a Stage that governs
-- it, and the tallies of those statuses. One statement serves both, because
-- both read the same entries: a pair's state is derived by one expression,
-- here, so the counts and the rows apply the same rules. They agree for the
-- same scope read from the same snapshot; two calls are two statements, and
-- the mirror may change between them.
--
-- The pairs are three arrays of equal length, zipped as in UpsertPromotion,
-- in the order the entries are returned in. A Stage is given by its id,
-- which the rows of its Promotions and rollouts carry, so a Stage deleted
-- and recreated under the same name starts with no history. Each Stage's
-- current Freight is given too, as two more arrays: the Stage's id and the
-- names of the Freight in its current collection, sorted in byte order and
-- joined with commas. A Stage with no current Freight is left out, and
-- none of its Targets is behind.
--
-- For each pair, in turn:
--
--   * last is the newest Promotion to the Target for the Stage, whatever
--     its phase.
--   * current is the newest of those that succeeded: last itself, when it
--     did. Its promotion_freight rows are the Freight the Target is on,
--     returned as three arrays of equal length, in origin order.
--   * A rollout says something the last Promotion does not when it names
--     the Target and came after that Promotion: it is numbered above the
--     rollout that created the Promotion or, for a Promotion created on
--     its own, was created after it. Such a rollout has no Promotion to the
--     Target, or that Promotion would be last. While it is unfinished the
--     Target is awaited. Once it has finished without succeeding, it never
--     attempted the Target. A rollout that names only some of a Stage's
--     Targets, such as a retry of those that failed, therefore says nothing
--     about the rest. A rollout that succeeded promoted to every Target it
--     names, so one that seems not to have is only ahead of the mirror, and
--     is not read.
--   * The Target is behind when the Freight it is on is not the Stage's
--     current Freight. With either unknown, it is not.
--
-- The state is the first of these that holds:
--
--   Promoting      last is Running.
--   Pending        last has yet to start, or the Target is awaited.
--   NotAttempted   a rollout never attempted the Target.
--   NeverPromoted  there has been no Promotion to the Target.
--   Failed         last did not succeed.
--   Behind         last succeeded and the Target is behind.
--   UpToDate       last succeeded.
--
-- With summarize unset, the rows are the entries: at most page_size of them,
-- in the order given, only those in one of the given states when any are
-- given. total is 1 and behind 1 or 0, and rollout_number is that of the
-- rollout a Pending or NotAttempted comes from when no Promotion says it,
-- else 0. With summarize set, the rows are
-- tallies instead, and only stage_id, state, total and behind mean anything:
-- for each Stage and state, the number of its entries in that state and how
-- many of those are behind; and, under an empty stage_id, the same for the
-- Targets themselves, each counted once under the state that comes first in
-- precedence among those its Stages report, and as behind if it is behind
-- for any of them.
--
-- Every lookup is a probe of an index per pair, and none reads further back
-- than it must, however long the history grows: last and current through
-- the Stage's Promotions to the Target, newest first, by
-- promotions_stage_target_created_at_idx; the unfinished rollouts through
-- promotion_rollouts_open_idx; and those that finished without succeeding
-- through promotion_rollouts_unsuccessful_idx, both only above the last
-- Promotion's number. The rollout predicates must match those indexes' word
-- for word.
-- name: ListTargetStatuses :many
WITH pairs AS (
    SELECT
        unnest(sqlc.arg(target_ids)::uuid[]) AS target_id,
        unnest(sqlc.arg(target_names)::text[]) AS target_name,
        unnest(sqlc.arg(stage_ids)::text[]) AS stage_id,
        generate_subscripts(sqlc.arg(target_ids)::uuid[], 1) AS position
),
stage_freight AS (
    SELECT
        unnest(sqlc.arg(freight_stage_ids)::text[]) AS stage_id,
        unnest(sqlc.arg(stage_freight)::text[]) AS freight
),
latest AS (
    SELECT
        pairs.target_id,
        pairs.target_name,
        pairs.stage_id,
        pairs.position,
        last.id AS last_id,
        last.phase AS last_phase,
        last.created_at AS last_created_at,
        last_rollout.number AS last_rollout_number
    FROM pairs
    LEFT JOIN LATERAL (
        SELECT
            promotions.id,
            promotions.phase,
            promotions.created_at,
            promotions.promotion_rollout_id
        FROM promotions
        WHERE promotions.stage_id = pairs.stage_id
            AND promotions.target_name = pairs.target_name
        ORDER BY promotions.created_at DESC, promotions.id DESC
        LIMIT 1
    ) AS last ON TRUE
    LEFT JOIN promotion_rollouts AS last_rollout
        ON last_rollout.id = last.promotion_rollout_id
),
observed AS MATERIALIZED (
    SELECT
        latest.*,
        CASE
            WHEN latest.last_id IS NULL THEN NULL
            WHEN latest.last_phase = 'Succeeded' THEN latest.last_id
            ELSE (
                SELECT promotions.id FROM promotions
                WHERE promotions.stage_id = latest.stage_id
                    AND promotions.target_name = latest.target_name
                    AND promotions.phase = 'Succeeded'
                ORDER BY promotions.created_at DESC, promotions.id DESC
                LIMIT 1
            )
        END AS current_id,
        awaited.number AS awaited_by,
        unreached.number AS not_attempted_by
    FROM latest
    LEFT JOIN LATERAL (
        SELECT promotion_rollouts.number FROM promotion_rollouts
        WHERE promotion_rollouts.stage_id = latest.stage_id
            AND promotion_rollouts.status_phase IN ('Pending', 'Running')
            AND promotion_rollouts.number > COALESCE(latest.last_rollout_number, 0)
            AND (
                latest.last_rollout_number IS NOT NULL
                OR promotion_rollouts.created_at > COALESCE(latest.last_created_at, '-infinity'::timestamptz)
            )
            AND EXISTS (
                SELECT 1 FROM promotion_rollout_targets
                WHERE promotion_rollout_targets.promotion_rollout_id = promotion_rollouts.id
                    AND promotion_rollout_targets.target_id = latest.target_id
            )
        ORDER BY promotion_rollouts.number
        LIMIT 1
    ) AS awaited ON TRUE
    LEFT JOIN LATERAL (
        SELECT promotion_rollouts.number FROM promotion_rollouts
        WHERE promotion_rollouts.stage_id = latest.stage_id
            AND promotion_rollouts.status_phase IN ('Failed', 'Errored', 'Aborted')
            AND promotion_rollouts.number > COALESCE(latest.last_rollout_number, 0)
            AND (
                latest.last_rollout_number IS NOT NULL
                OR promotion_rollouts.created_at > COALESCE(latest.last_created_at, '-infinity'::timestamptz)
            )
            AND EXISTS (
                SELECT 1 FROM promotion_rollout_targets
                WHERE promotion_rollout_targets.promotion_rollout_id = promotion_rollouts.id
                    AND promotion_rollout_targets.target_id = latest.target_id
            )
        ORDER BY promotion_rollouts.number DESC
        LIMIT 1
    ) AS unreached ON TRUE
),
compared AS (
    SELECT
        observed.*,
        COALESCE(on_target.freight <> stage_freight.freight, FALSE) AS behind
    FROM observed
    LEFT JOIN stage_freight ON stage_freight.stage_id = observed.stage_id
    LEFT JOIN LATERAL (
        SELECT string_agg(
            promotion_freight.freight_name, ','
            ORDER BY promotion_freight.freight_name COLLATE "C"
        ) AS freight
        FROM promotion_freight
        WHERE promotion_freight.promotion_id = observed.current_id
    ) AS on_target ON TRUE
),
entries AS (
    SELECT
        compared.*,
        CASE
            WHEN compared.last_phase = 'Running' THEN 'Promoting'
            WHEN compared.last_phase IN ('Pending', '')
                OR compared.awaited_by IS NOT NULL THEN 'Pending'
            WHEN compared.not_attempted_by IS NOT NULL THEN 'NotAttempted'
            WHEN compared.last_id IS NULL THEN 'NeverPromoted'
            WHEN compared.last_phase <> 'Succeeded' THEN 'Failed'
            WHEN compared.behind THEN 'Behind'
            ELSE 'UpToDate'
        END AS state
    FROM compared
),
page AS (
    SELECT * FROM entries
    WHERE NOT sqlc.arg(summarize)::boolean
        AND (
            cardinality(sqlc.arg(states)::text[]) = 0
            OR entries.state = ANY(sqlc.arg(states)::text[])
        )
    ORDER BY entries.position
    LIMIT sqlc.arg(page_size)
),
by_target AS (
    SELECT
        entries.target_id,
        min(array_position(sqlc.arg(precedence)::text[], entries.state)) AS rank,
        bool_or(entries.behind) AS behind
    FROM entries
    WHERE sqlc.arg(summarize)::boolean
    GROUP BY entries.target_id
)
SELECT
    page.stage_id::text AS stage_id,
    page.target_id::uuid AS target_id,
    page.position::bigint AS position,
    page.state::text AS state,
    1::bigint AS total,
    page.behind::int::bigint AS behind,
    (page.state IN ('Promoting', 'Pending'))::boolean AS in_flight,
    COALESCE(last.id, '')::text AS last_id,
    COALESCE(last.name, '')::text AS last_name,
    COALESCE(NULLIF(last.phase, ''), 'Pending')::text AS last_phase,
    COALESCE(last.freight_name, '')::text AS last_freight_name,
    last.created_at AS last_created_at,
    last.finished_at AS last_finished_at,
    COALESCE(page.last_rollout_number, 0)::bigint AS last_rollout_number,
    COALESCE(freight.origin_kinds, '{}')::text[] AS origin_kinds,
    COALESCE(freight.origin_names, '{}')::text[] AS origin_names,
    COALESCE(freight.freight_names, '{}')::text[] AS freight_names,
    (
        CASE
            WHEN page.state = 'NotAttempted' THEN page.not_attempted_by
            WHEN page.state = 'Pending' AND page.last_phase IS DISTINCT FROM 'Pending'
                AND page.last_phase IS DISTINCT FROM '' THEN page.awaited_by
            ELSE 0
        END
    )::bigint AS rollout_number
FROM page
LEFT JOIN promotions AS last ON last.id = page.last_id
LEFT JOIN LATERAL (
    SELECT
        array_agg(promotion_freight.origin_kind ORDER BY promotion_freight.origin_kind, promotion_freight.origin_name) AS origin_kinds,
        array_agg(promotion_freight.origin_name ORDER BY promotion_freight.origin_kind, promotion_freight.origin_name) AS origin_names,
        array_agg(promotion_freight.freight_name ORDER BY promotion_freight.origin_kind, promotion_freight.origin_name) AS freight_names
    FROM promotion_freight
    WHERE promotion_freight.promotion_id = page.current_id
) AS freight ON TRUE
UNION ALL
SELECT
    entries.stage_id, '00000000-0000-0000-0000-000000000000', 0, entries.state,
    count(*), count(*) FILTER (WHERE entries.behind),
    FALSE, '', '', '', '', NULL, NULL, 0, '{}', '{}', '{}', 0
FROM entries
WHERE sqlc.arg(summarize)::boolean
GROUP BY entries.stage_id, entries.state
UNION ALL
SELECT
    '', '00000000-0000-0000-0000-000000000000', 0, (sqlc.arg(precedence)::text[])[by_target.rank],
    count(*), count(*) FILTER (WHERE by_target.behind),
    FALSE, '', '', '', '', NULL, NULL, 0, '{}', '{}', '{}', 0
FROM by_target
GROUP BY by_target.rank
ORDER BY position, stage_id, state;
