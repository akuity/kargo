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

-- Counts the Project's Targets, and how many of them are among the names
-- given, without reading any Target's labels or params.
-- name: CountTargets :one
SELECT
    count(*) AS total,
    count(*) FILTER (WHERE name = ANY(sqlc.arg(names)::text[])) AS among
FROM targets
WHERE project_name = sqlc.arg(project_name);

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

-- Returns, for each of the Project's Targets, or only the named one, and each
-- Stage that has successfully promoted to it, the Freight of that Stage's most
-- recent successful Promotion to it: one row per origin. Stages are those the
-- database mirrors, so a deleted Stage's Promotions say nothing. Promotions
-- name their Target, so a Target recreated under the same name inherits the
-- record of what was promoted to its predecessor, which, under the same name,
-- is almost always the same destination.
-- name: ListTargetCurrentFreight :many
SELECT
    targets.id AS target_id,
    stages.name AS stage_name,
    promotion_freight.origin_kind,
    promotion_freight.origin_name,
    promotion_freight.freight_name
FROM targets
JOIN stages ON stages.project_name = targets.project_name
CROSS JOIN LATERAL (
    SELECT promotions.id
    FROM promotions
    WHERE promotions.project_name = targets.project_name
        AND promotions.target_name = targets.name
        AND promotions.stage_name = stages.name
        AND promotions.phase = 'Succeeded'
        AND promotions.finished_at IS NOT NULL
    ORDER BY promotions.finished_at DESC, promotions.id DESC
    LIMIT 1
) AS succeeded
JOIN promotion_freight ON promotion_freight.promotion_id = succeeded.id
WHERE targets.project_name = sqlc.arg(project_name)
    AND (sqlc.narg(target_name)::text IS NULL OR targets.name = sqlc.narg(target_name))
ORDER BY
    targets.name,
    stages.name,
    promotion_freight.origin_kind,
    promotion_freight.origin_name;

-- Returns, for each of the Project's Targets and each Stage that has promoted
-- to it, that Stage's most recent Promotion to it, whatever its phase.
-- name: ListTargetLatestPromotions :many
SELECT
    targets.id AS target_id,
    latest.name,
    latest.stage_name,
    latest.freight_name,
    latest.phase,
    latest.created_at,
    latest.finished_at
FROM targets
JOIN stages ON stages.project_name = targets.project_name
CROSS JOIN LATERAL (
    SELECT
        promotions.name,
        promotions.stage_name,
        promotions.freight_name,
        promotions.phase,
        promotions.created_at,
        promotions.finished_at
    FROM promotions
    WHERE promotions.project_name = targets.project_name
        AND promotions.target_name = targets.name
        AND promotions.stage_name = stages.name
    ORDER BY promotions.created_at DESC, promotions.id DESC
    LIMIT 1
) AS latest
WHERE targets.project_name = sqlc.arg(project_name)
ORDER BY targets.name, latest.stage_name;

-- Returns the Promotions to the named Target, newest first, from every Stage
-- and including those Kubernetes no longer has. A Target is named, not
-- referenced, so this is the history of the name.
-- name: ListTargetPromotions :many
SELECT * FROM promotions
WHERE project_name = sqlc.arg(project_name) AND target_name = sqlc.arg(target_name)::text
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(max_rows)::int;
