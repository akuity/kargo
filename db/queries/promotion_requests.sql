-- PromotionRequests are addressed by Project name and request name, and by
-- id once one is known. Their Stage, Freight and Targets are names too.

-- Numbers the request after the Stage's last one, creating the Stage's row on
-- its first request. The Stage's row stays locked until the transaction ends,
-- which serializes the Stage's requests; a create that fails rolls the counter
-- back with it, so numbers have no gaps.
-- name: CreatePromotionRequest :one
WITH stage AS (
    INSERT INTO stages (project_name, name, last_promotion_request_number)
    VALUES (sqlc.arg(project_name)::text, sqlc.arg(stage_name)::text, 1)
    ON CONFLICT (project_name, name) DO UPDATE SET
        last_promotion_request_number = stages.last_promotion_request_number + 1
    RETURNING last_promotion_request_number
)
INSERT INTO promotion_requests (
    project_name, name, stage_name, number, freight_name, update_strategy, created_by
)
SELECT
    sqlc.arg(project_name)::text,
    sqlc.arg(name)::text,
    sqlc.arg(stage_name)::text,
    stage.last_promotion_request_number,
    sqlc.arg(freight_name)::text,
    sqlc.arg(update_strategy)::jsonb,
    sqlc.arg(created_by)::text
FROM stage
RETURNING *;

-- name: GetPromotionRequest :one
SELECT * FROM promotion_requests
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- name: GetPromotionRequestByID :one
SELECT * FROM promotion_requests WHERE id = $1;

-- Locks the request's row until the transaction ends, so that what it
-- returns is still true when the caller writes.
-- name: GetPromotionRequestByIDForUpdate :one
SELECT * FROM promotion_requests WHERE id = $1 FOR UPDATE;

-- Numbers are per Stage, so a Project's requests are ordered by creation time.
-- name: ListPromotionRequests :many
SELECT * FROM promotion_requests
WHERE project_name = sqlc.arg(project_name)
ORDER BY created_at, stage_name, number;

-- name: ListPromotionRequestsByStage :many
SELECT * FROM promotion_requests
WHERE project_name = sqlc.arg(project_name) AND stage_name = sqlc.arg(stage_name)
ORDER BY number;

-- The requests that fan out to the named Target, newest first.
-- name: ListPromotionRequestsByTarget :many
SELECT promotion_requests.* FROM promotion_requests
JOIN promotion_request_targets
    ON promotion_request_targets.promotion_request_id = promotion_requests.id
WHERE promotion_requests.project_name = sqlc.arg(project_name)
    AND promotion_request_targets.target_name = sqlc.arg(target_name)
ORDER BY promotion_requests.created_at DESC, promotion_requests.number DESC;

-- The ids of the open requests, paged by id, for a reconciler's resync.
-- name: ListOpenPromotionRequestIDs :many
SELECT id FROM promotion_requests
WHERE phase IN ('Pending', 'Running') AND id > sqlc.arg(after_id)
ORDER BY id
LIMIT sqlc.arg(row_limit);

-- updated_at doubles as the row's version, so it must grow with every write
-- to a row even across a clock step or two writes in one microsecond.
-- name: UpdatePromotionRequestStatus :one
UPDATE promotion_requests SET
    phase = sqlc.arg(phase)::text,
    message = sqlc.arg(message)::text,
    started_at = sqlc.narg(started_at)::timestamptz,
    finished_at = sqlc.narg(finished_at)::timestamptz,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE id = $1
RETURNING *;

-- name: DeletePromotionRequest :execrows
DELETE FROM promotion_requests
WHERE project_name = sqlc.arg(project_name) AND name = sqlc.arg(name);

-- Adds a Target to the request's fan-out. A Target already present is left
-- alone and nothing is returned.
-- name: AddPromotionRequestTarget :one
INSERT INTO promotion_request_targets (promotion_request_id, target_name)
VALUES (sqlc.arg(promotion_request_id), sqlc.arg(target_name))
ON CONFLICT DO NOTHING
RETURNING *;

-- name: ListPromotionRequestTargets :many
SELECT * FROM promotion_request_targets
WHERE promotion_request_id = ANY(sqlc.arg(promotion_request_ids)::uuid[])
ORDER BY promotion_request_id, target_name;

-- The Targets a request fans out to, as they are now. A Target deleted since
-- the request was made is absent; ListPromotionRequestTargets still has it.
-- name: ListTargetsOfPromotionRequest :many
SELECT targets.* FROM promotion_request_targets
JOIN promotion_requests
    ON promotion_requests.id = promotion_request_targets.promotion_request_id
JOIN targets
    ON targets.project_name = promotion_requests.project_name
    AND targets.name = promotion_request_targets.target_name
WHERE promotion_request_targets.promotion_request_id = $1
ORDER BY targets.name;

-- name: UpdatePromotionRequestTarget :execrows
UPDATE promotion_request_targets SET
    promotion = sqlc.arg(promotion)::text,
    phase = sqlc.arg(phase)::text
WHERE promotion_request_id = sqlc.arg(promotion_request_id)
    AND target_name = sqlc.arg(target_name);
