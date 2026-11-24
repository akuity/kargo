-- Promotion rows are inserted and updated here, never deleted: a row is kept
-- past its Promotion's deletion as the record of what it did, and goes only
-- with its Stage, by cascade from the stages table. See the migrations that
-- create the table and add the cascade.

-- The Promotion's FreightCollection, if it has one yet, is recorded in the same
-- statement, one row per origin, given as three arrays of equal length. A
-- Promotion builds its collection once, so a row that exists is kept.
--
-- A running Promotion's status changes often while nothing mirrored here does,
-- so an update that would change nothing is skipped, synced_at included. The
-- collection is recorded under the id the Promotion's row already has, or
-- the one given for a new row, rather than by the upsert's RETURNING, which
-- is empty when the update is skipped: a collection that arrives with no
-- other change is still recorded.
--
-- The row is found by the Promotion's Project and name. The id given is used
-- only for a new row; a row keeps the id it was first written with. A row
-- whose creation time differs belongs to another Promotion that had the same
-- name. Names are never given twice, so this should not happen; if it does,
-- that record is left as it is and name_taken reports it.
--
-- The row references its Stage's mirrored row, found by Project and Stage
-- name, so that it is deleted with the Stage. When the Promotion names the
-- Stage that owns it, by owner_stage_id, the row found must be that Stage's:
-- a Stage deleted and recreated under the same name is a new row, and a
-- Promotion of the old one, still in Kubernetes until it is collected, must
-- not be recorded under the new one. A Stage the mirror has not recorded
-- yet, or one that has been replaced, has no row to find, and the write
-- fails on the column's NOT NULL constraint. The reference is written once:
-- a row never moves to another Stage, since a Stage's replacement deletes
-- the rows of the old one.
--
-- The PromotionRollout the Promotion was created by is recorded only if it
-- is one of the same Stage's; an id naming no such rollout is recorded as
-- none, so that a rollout's deletion never takes another Stage's history
-- with it. Once recorded it is kept, whatever the Promotion later claims.
-- name: UpsertPromotion :one
WITH stage AS (
    SELECT (
        SELECT stages.id FROM stages
        WHERE stages.project_name = sqlc.arg(project_name)
            AND stages.name = sqlc.arg(stage_name)
            AND (
                sqlc.narg(owner_stage_id)::text IS NULL
                OR stages.id = sqlc.narg(owner_stage_id)::text
            )
    ) AS id
),
existing AS (
    SELECT
        (
            SELECT promotions.id FROM promotions
            WHERE promotions.project_name = sqlc.arg(project_name)
                AND promotions.name = sqlc.arg(name)
        ) AS id,
        EXISTS (
            SELECT 1 FROM promotions
            WHERE promotions.project_name = sqlc.arg(project_name)
                AND promotions.name = sqlc.arg(name)
                AND promotions.created_at <> sqlc.arg(created_at)
        ) AS name_taken
),
promotion AS (
    INSERT INTO promotions (
        id,
        project_name,
        name,
        stage_id,
        stage_name,
        freight_name,
        target_name,
        promotion_rollout_id,
        phase,
        created_at,
        finished_at
    )
    VALUES (
        sqlc.arg(id),
        sqlc.arg(project_name),
        sqlc.arg(name),
        (SELECT stage.id FROM stage),
        sqlc.arg(stage_name),
        sqlc.arg(freight_name),
        sqlc.narg(target_name),
        (
            SELECT promotion_rollouts.id FROM promotion_rollouts, stage
            WHERE promotion_rollouts.stage_id = stage.id
                AND promotion_rollouts.id = sqlc.narg(promotion_rollout_id)
        ),
        sqlc.arg(phase),
        sqlc.arg(created_at),
        sqlc.narg(finished_at)
    )
    ON CONFLICT (project_name, name) DO UPDATE SET
        stage_name = EXCLUDED.stage_name,
        freight_name = EXCLUDED.freight_name,
        target_name = EXCLUDED.target_name,
        promotion_rollout_id = COALESCE(promotions.promotion_rollout_id, EXCLUDED.promotion_rollout_id),
        phase = EXCLUDED.phase,
        finished_at = EXCLUDED.finished_at,
        synced_at = CURRENT_TIMESTAMP
    WHERE promotions.created_at = EXCLUDED.created_at
        AND (
            promotions.stage_name,
            promotions.freight_name,
            promotions.target_name,
            promotions.promotion_rollout_id,
            promotions.phase,
            promotions.finished_at
        ) IS DISTINCT FROM (
            EXCLUDED.stage_name,
            EXCLUDED.freight_name,
            EXCLUDED.target_name,
            COALESCE(promotions.promotion_rollout_id, EXCLUDED.promotion_rollout_id),
            EXCLUDED.phase,
            EXCLUDED.finished_at
        )
),
freight AS (
    INSERT INTO promotion_freight (promotion_id, origin_kind, origin_name, freight_name)
    -- Set-returning functions in one select list advance together, which zips
    -- the arrays.
    SELECT
        COALESCE(existing.id, sqlc.arg(id)),
        unnest(sqlc.arg(origin_kinds)::text[]),
        unnest(sqlc.arg(origin_names)::text[]),
        unnest(sqlc.arg(freight_names)::text[])
    FROM existing
    WHERE NOT existing.name_taken
    ON CONFLICT (promotion_id, origin_kind, origin_name) DO NOTHING
)
SELECT name_taken FROM existing;

-- The table grows without bound, so the mirror never lists all of it: it
-- asks only after the rows of the Promotions Kubernetes currently has, given
-- as two arrays of equal length, zipped as in UpsertPromotion. Each key is
-- looked up on its own, through the unique index: a join would let the
-- planner scan the whole table instead when it expects many keys, and the
-- LIMIT, harmless since a key matches at most one row, keeps the lookup from
-- being flattened into one.
-- name: ListPromotionsByKey :many
WITH keys AS (
    SELECT
        unnest(sqlc.arg(project_names)::text[]) AS project_name,
        unnest(sqlc.arg(names)::text[]) AS name
)
SELECT promotions.*
FROM keys
CROSS JOIN LATERAL (
    SELECT * FROM promotions
    WHERE promotions.project_name = keys.project_name
        AND promotions.name = keys.name
    LIMIT 1
) AS promotions
ORDER BY promotions.project_name, promotions.name;

-- Rows of Promotions that have not finished. The resync compares them with
-- Kubernetes so that a Promotion deleted before it finished, whose deletion
-- the mirror may not have seen, is still recorded as finished.
-- name: ListUnfinishedPromotions :many
SELECT * FROM promotions
WHERE finished_at IS NULL
ORDER BY project_name, name;

-- Records a Promotion that was deleted before it finished as finished in the
-- given phase, at the time its deletion was seen. A row already finished is
-- left alone: it is the record of what the Promotion did.
-- name: FinishDeletedPromotion :execrows
UPDATE promotions SET
    phase = sqlc.arg(phase),
    finished_at = CURRENT_TIMESTAMP,
    synced_at = CURRENT_TIMESTAMP
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND finished_at IS NULL;
