-- +goose Up

-- Serve finding, for each Target and each Stage that has promoted to it, the
-- most recent Promotion and the most recent successful one: one index probe
-- per Target and Stage, however long the history grows.
CREATE INDEX promotions_target_stage_created_at_idx
    ON promotions (project_name, target_name, stage_name, created_at DESC, id DESC)
    WHERE target_name IS NOT NULL;
CREATE INDEX promotions_target_stage_succeeded_idx
    ON promotions (project_name, target_name, stage_name, finished_at DESC, id DESC)
    WHERE target_name IS NOT NULL AND phase = 'Succeeded' AND finished_at IS NOT NULL;

-- +goose Down
DROP INDEX promotions_target_stage_succeeded_idx;
DROP INDEX promotions_target_stage_created_at_idx;
