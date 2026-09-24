-- name: DeleteReplacedWarehouse :exec
UPDATE warehouses SET deleted_at = CURRENT_TIMESTAMP
WHERE project_id = sqlc.arg(project_id)
    AND name = sqlc.arg(name) AND id <> sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpsertWarehouse :exec
INSERT INTO warehouses (id, project_id, name, created_at)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
    project_id = EXCLUDED.project_id,
    name = EXCLUDED.name,
    created_at = EXCLUDED.created_at,
    synced_at = CURRENT_TIMESTAMP,
    deleted_at = NULL;

-- name: DeleteWarehouseByName :exec
UPDATE warehouses SET deleted_at = CURRENT_TIMESTAMP
FROM projects
WHERE warehouses.project_id = projects.id
    AND projects.name = sqlc.arg(project_name)
    AND warehouses.name = sqlc.arg(name) AND warehouses.deleted_at IS NULL;

-- name: ListWarehouses :many
SELECT * FROM warehouses WHERE deleted_at IS NULL ORDER BY id;

-- name: DeleteWarehousesByID :exec
UPDATE warehouses SET deleted_at = CURRENT_TIMESTAMP
WHERE id = ANY($1::text[]) AND deleted_at IS NULL;
