-- name: DeleteReplacedFreight :exec
UPDATE freight SET deleted_at = CURRENT_TIMESTAMP
WHERE project_id = sqlc.arg(project_id)
    AND name = sqlc.arg(name) AND id <> sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpsertFreight :exec
INSERT INTO freight (id, project_id, warehouse_id, name, alias, discovered_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    alias = EXCLUDED.alias,
    discovered_at = EXCLUDED.discovered_at,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP,
    deleted_at = NULL;

-- name: GetFreightByID :one
SELECT * FROM freight WHERE id = $1;

-- name: GetFreightWarehouseID :one
SELECT warehouse_id FROM freight WHERE id = $1;

-- name: DeleteFreightByName :exec
UPDATE freight SET deleted_at = CURRENT_TIMESTAMP
FROM projects
WHERE freight.project_id = projects.id
    AND projects.name = sqlc.arg(project_name)
    AND freight.name = sqlc.arg(name) AND freight.deleted_at IS NULL;

-- name: ListFreight :many
SELECT * FROM freight WHERE deleted_at IS NULL ORDER BY id;

-- name: DeleteFreightByIDs :exec
UPDATE freight SET deleted_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::text[]) AND deleted_at IS NULL;
