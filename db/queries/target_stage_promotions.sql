-- Rows are written only alongside Promotion rows, by UpsertPromotion and
-- FinishDeletedPromotion; see the migration that creates the table.

-- The Project's pairs, or only those of the named Target: for each Stage
-- and Target it has promoted to, the latest Promotion and the latest
-- successful one.
-- name: ListTargetStagePromotions :many
SELECT * FROM target_stage_promotions
WHERE project_name = sqlc.arg(project_name)
    AND (sqlc.narg(target_name)::text IS NULL OR target_name = sqlc.narg(target_name))
ORDER BY target_name, stage_name;
