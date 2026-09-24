-- name: ListTargets :many
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name)
ORDER BY targets.name;

-- name: GetTarget :one
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name) AND targets.name = sqlc.arg(name);

-- name: ListTargetsByName :many
SELECT targets.* FROM targets
JOIN projects ON projects.id = targets.project_id
WHERE projects.name = sqlc.arg(project_name)
    AND targets.name = ANY(sqlc.arg(names)::text[])
ORDER BY targets.name;
