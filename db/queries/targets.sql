-- Targets are addressed by Project name and Target name everywhere, in the
-- database as outside it.

-- name: CreateTarget :one
INSERT INTO targets (project_name, name, labels, params, target_set_name)
VALUES (
    sqlc.arg(project_name)::text,
    sqlc.arg(name)::text,
    sqlc.arg(labels)::jsonb,
    sqlc.arg(params)::jsonb,
    sqlc.arg(target_set_name)::text
)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM targets
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- Counts the Project's Targets, and how many of them are healthy, without
-- reading any Target's labels or params. A Target's status does not record
-- its health yet, so a Target counts as healthy when at least one Stage has
-- promoted to it and the latest promotion to it from every Stage that did so
-- succeeded.
--
-- A Stage's latest promotion to a Target is made by the Stage's latest
-- PromotionRollout: the one it is promoting through, if any, else the newest
-- that has finished -- the two that ListLatestPromotionRollouts reads, and
-- read the same way. Of that rollout's Promotions to the Target, the newest
-- counts, so a Target promoted to again after a failure reports the retry.
-- A Target the rollout has no Promotion to is Pending while the rollout
-- runs, and takes the rollout's own phase once it has ended: a rollout that
-- ended without fanning out ended every one of its Targets that way.
--
-- Only target-aware Stages, those the mirror records with selectors, are
-- read: a Stage converted back to classic says nothing about Targets. Each
-- latest rollout's Promotions are read once, through the index on their
-- rollout, so the work grows with those rollouts' fan-out rather than with
-- the Project's history. Targets are joined within the Project too, so no
-- other Project's rows are read, though Target names repeat across Projects.
-- name: CountTargets :one
WITH latest AS (
    SELECT DISTINCT ON (stages.name) latest.id, latest.status_phase
    FROM stages
    CROSS JOIN LATERAL (
        (
            SELECT promotion_rollouts.id, promotion_rollouts.status_phase
            FROM promotion_rollouts
            WHERE promotion_rollouts.project_name = stages.project_name
                AND promotion_rollouts.stage_name = stages.name
                AND promotion_rollouts.status_phase IN ('Pending', 'Running')
            ORDER BY promotion_rollouts.status_phase = 'Running' DESC, promotion_rollouts.number
            LIMIT 1
        )
        UNION ALL
        (
            SELECT promotion_rollouts.id, promotion_rollouts.status_phase
            FROM promotion_rollouts
            WHERE promotion_rollouts.project_name = stages.project_name
                AND promotion_rollouts.stage_name = stages.name
                AND promotion_rollouts.status_phase NOT IN ('Pending', 'Running')
            ORDER BY promotion_rollouts.number DESC
            LIMIT 1
        )
    ) AS latest
    WHERE stages.project_name = sqlc.arg(project_name)
        AND stages.target_selectors IS NOT NULL
    ORDER BY stages.name, latest.status_phase IN ('Pending', 'Running') DESC
),
newest AS (
    SELECT DISTINCT ON (promotions.promotion_rollout_id, promotions.target_name)
        promotions.promotion_rollout_id, promotions.target_name, promotions.phase
    FROM latest
    JOIN promotions ON promotions.promotion_rollout_id = latest.id
    ORDER BY promotions.promotion_rollout_id, promotions.target_name,
        promotions.created_at DESC, promotions.id DESC
),
target_phases AS (
    SELECT
        promotion_rollout_targets.target_id,
        COALESCE(
            newest.phase,
            CASE
                WHEN latest.status_phase IN ('Succeeded', 'Failed', 'Errored') THEN latest.status_phase
                ELSE 'Pending'
            END
        ) AS phase
    FROM latest
    JOIN promotion_rollout_targets ON promotion_rollout_targets.promotion_rollout_id = latest.id
    JOIN targets ON targets.id = promotion_rollout_targets.target_id
        AND targets.project_name = sqlc.arg(project_name)
    LEFT JOIN newest ON newest.promotion_rollout_id = latest.id
        AND newest.target_name = targets.name
)
SELECT
    (SELECT count(*) FROM targets WHERE targets.project_name = sqlc.arg(project_name)) AS total,
    (
        SELECT count(*) FROM (
            SELECT target_phases.target_id FROM target_phases
            GROUP BY target_phases.target_id
            HAVING bool_and(target_phases.phase = 'Succeeded')
        ) AS healthy
    ) AS healthy;

-- name: ListTargets :many
SELECT * FROM targets
WHERE project_name = sqlc.arg(project_name)
ORDER BY name;

-- updated_at doubles as the row's version, so it must grow with every write
-- to a row even across a clock step or two writes in one microsecond. The
-- TargetSet name never changes, so an update asserts it rather than setting
-- it: a row managed by another TargetSet is left alone. A non-null id or
-- updated_at is a further precondition; a row that fails one is left alone
-- and nothing is returned.
-- name: UpdateTarget :one
UPDATE targets SET
    labels = sqlc.arg(labels)::jsonb,
    params = sqlc.arg(params)::jsonb,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND target_set_name = sqlc.arg(target_set_name)::text
    AND (sqlc.narg(id)::uuid IS NULL OR id = sqlc.narg(id))
    AND (
        sqlc.narg(updated_at)::timestamptz IS NULL
        OR updated_at = sqlc.narg(updated_at)
    )
RETURNING *;

-- name: DeleteTarget :execrows
DELETE FROM targets
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);
