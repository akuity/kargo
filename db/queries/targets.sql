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
