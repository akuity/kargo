-- Promotion rows are only ever inserted and updated; see the migration that
-- creates the table. There is no delete.

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
-- name: UpsertPromotion :one
WITH existing AS (
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
    ON CONFLICT (project_name, name) DO UPDATE SET
        stage_name = EXCLUDED.stage_name,
        freight_name = EXCLUDED.freight_name,
        target_name = EXCLUDED.target_name,
        phase = EXCLUDED.phase,
        finished_at = EXCLUDED.finished_at,
        synced_at = CURRENT_TIMESTAMP
    WHERE promotions.created_at = EXCLUDED.created_at
        AND (
            promotions.stage_name,
            promotions.freight_name,
            promotions.target_name,
            promotions.phase,
            promotions.finished_at
        ) IS DISTINCT FROM (
            EXCLUDED.stage_name,
            EXCLUDED.freight_name,
            EXCLUDED.target_name,
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
-- as two arrays of equal length, zipped as in UpsertPromotion.
-- name: ListPromotionsByKey :many
SELECT * FROM promotions
WHERE (project_name, name) IN (
    SELECT unnest(sqlc.arg(project_names)::text[]), unnest(sqlc.arg(names)::text[])
)
ORDER BY project_name, name;
