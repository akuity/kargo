-- Targets are addressed by Project name and Target name everywhere, in the
-- database as outside it.

-- name: CreateTarget :one
INSERT INTO targets (project_name, name, labels, params)
VALUES (
    sqlc.arg(project_name)::text,
    sqlc.arg(name)::text,
    sqlc.arg(labels)::jsonb,
    sqlc.arg(params)::jsonb
)
RETURNING *;

-- name: GetTarget :one
SELECT * FROM targets
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- name: ListTargets :many
SELECT * FROM targets
WHERE project_name = sqlc.arg(project_name)
ORDER BY name;

-- updated_at doubles as the row's version, so it must grow with every write
-- to a row even across a clock step or two writes in one microsecond. A
-- non-null id or updated_at is a precondition; a row that fails one is left
-- alone and nothing is returned.
-- name: UpdateTarget :one
UPDATE targets SET
    labels = sqlc.arg(labels)::jsonb,
    params = sqlc.arg(params)::jsonb,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND name = sqlc.arg(name)
    AND (sqlc.narg(id)::uuid IS NULL OR id = sqlc.narg(id))
    AND (
        sqlc.narg(updated_at)::timestamptz IS NULL
        OR updated_at = sqlc.narg(updated_at)
    )
RETURNING *;

-- name: DeleteTarget :execrows
DELETE FROM targets
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);
