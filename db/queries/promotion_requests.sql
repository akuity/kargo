-- The Stage and Freight a PromotionRequest refers to are resolved from their
-- names when the row is inserted. Nothing is inserted when any of the Project,
-- Stage or Freight has not been mirrored yet.
-- name: CreatePromotionRequest :one
INSERT INTO promotion_requests (project_id, stage_id, freight_id, name, created_by)
SELECT projects.id, stages.id, freight.id, sqlc.arg(name)::text, sqlc.arg(created_by)::text
FROM projects
JOIN stages ON stages.project_id = projects.id AND stages.name = sqlc.arg(stage)
JOIN freight ON freight.project_id = projects.id
    AND freight.name = sqlc.arg(freight) AND freight.deleted_at IS NULL
WHERE projects.name = sqlc.arg(project_name)
RETURNING *;

-- name: GetPromotionRequest :one
SELECT sqlc.embed(promotion_requests),
    projects.name AS project_name, stages.name AS stage, freight.name AS freight
FROM promotion_requests
JOIN projects ON projects.id = promotion_requests.project_id
JOIN stages ON stages.id = promotion_requests.stage_id
JOIN freight ON freight.id = promotion_requests.freight_id
WHERE projects.name = sqlc.arg(project_name) AND promotion_requests.name = sqlc.arg(name);

-- name: GetPromotionRequestByID :one
SELECT sqlc.embed(promotion_requests),
    projects.name AS project_name, stages.name AS stage, freight.name AS freight
FROM promotion_requests
JOIN projects ON projects.id = promotion_requests.project_id
JOIN stages ON stages.id = promotion_requests.stage_id
JOIN freight ON freight.id = promotion_requests.freight_id
WHERE promotion_requests.id = $1;

-- Locks the request's row, and only that row, until the transaction ends, so
-- that what it returns is still true when the caller writes.
-- name: GetPromotionRequestByIDForUpdate :one
SELECT sqlc.embed(promotion_requests),
    projects.name AS project_name, stages.name AS stage, freight.name AS freight
FROM promotion_requests
JOIN projects ON projects.id = promotion_requests.project_id
JOIN stages ON stages.id = promotion_requests.stage_id
JOIN freight ON freight.id = promotion_requests.freight_id
WHERE promotion_requests.id = $1
FOR UPDATE OF promotion_requests;

-- name: ListPromotionRequests :many
SELECT sqlc.embed(promotion_requests),
    projects.name AS project_name, stages.name AS stage, freight.name AS freight
FROM promotion_requests
JOIN projects ON projects.id = promotion_requests.project_id
JOIN stages ON stages.id = promotion_requests.stage_id
JOIN freight ON freight.id = promotion_requests.freight_id
WHERE projects.name = sqlc.arg(project_name)
ORDER BY promotion_requests.seq;

-- name: ListPromotionRequestsByStage :many
SELECT sqlc.embed(promotion_requests),
    projects.name AS project_name, stages.name AS stage, freight.name AS freight
FROM promotion_requests
JOIN projects ON projects.id = promotion_requests.project_id
JOIN stages ON stages.id = promotion_requests.stage_id
JOIN freight ON freight.id = promotion_requests.freight_id
WHERE projects.name = sqlc.arg(project_name) AND stages.name = sqlc.arg(stage)
ORDER BY promotion_requests.seq;

-- name: PromotionRequestExists :one
SELECT EXISTS (
    SELECT 1 FROM promotion_requests
    JOIN projects ON projects.id = promotion_requests.project_id
    JOIN stages ON stages.id = promotion_requests.stage_id
    JOIN freight ON freight.id = promotion_requests.freight_id
    WHERE projects.name = sqlc.arg(project_name)
        AND stages.name = sqlc.arg(stage)
        AND freight.name = sqlc.arg(freight)
);

-- The ids of the open requests, paged by id, for the reconciler's resync.
-- name: ListOpenPromotionRequestIDs :many
SELECT id FROM promotion_requests
WHERE phase IN ('Pending', 'Running') AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(row_limit);

-- name: UpdatePromotionRequestStatus :execrows
UPDATE promotion_requests SET
    phase = $2,
    message = $3,
    started_at = $4,
    finished_at = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: InsertPromotionRequestTarget :exec
INSERT INTO promotion_request_targets (promotion_request_id, target_id, ordinal)
VALUES ($1, $2, $3);

-- name: ListPromotionRequestTargets :many
SELECT t.promotion_request_id, t.target_id, targets.name, t.ordinal, t.promotion, t.phase
FROM promotion_request_targets t
JOIN targets ON targets.id = t.target_id
WHERE t.promotion_request_id = ANY(sqlc.arg(promotion_request_ids)::uuid[])
ORDER BY t.promotion_request_id, t.ordinal;

-- name: UpdatePromotionRequestTarget :exec
UPDATE promotion_request_targets SET promotion = $3, phase = $4
WHERE promotion_request_id = $1 AND target_id = $2;
