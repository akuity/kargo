-- Targets are addressed by Project name and Target name everywhere outside
-- the database, so every query resolves the Project through its mirrored
-- row. A write that finds no such row reports it by returning nothing.

-- name: CreateTarget :one
INSERT INTO targets (project_id, name, labels, params)
SELECT projects.id, sqlc.arg(name)::text, sqlc.arg(labels)::jsonb, sqlc.arg(params)::jsonb
FROM projects
WHERE projects.name = sqlc.arg(project_name)
RETURNING *;

-- name: GetTarget :one
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name) AND targets.name = sqlc.arg(name);

-- Locks only the Target row: locking the joined Project row too would block
-- the mirror's own writes to it.
-- name: GetTargetForUpdate :one
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name) AND targets.name = sqlc.arg(name)
FOR UPDATE OF targets;

-- name: ListTargets :many
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name)
ORDER BY targets.name;

-- updated_at doubles as the resource version, so it must grow with every
-- write to a row even across a clock step or two writes in one microsecond.
-- name: UpdateTarget :one
UPDATE targets SET
    labels = sqlc.arg(labels)::jsonb,
    params = sqlc.arg(params)::jsonb,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteTarget :execrows
DELETE FROM targets USING projects
WHERE targets.project_id = projects.id
    AND projects.name = sqlc.arg(project_name)
    AND targets.name = sqlc.arg(name);
