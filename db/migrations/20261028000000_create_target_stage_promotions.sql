-- +goose Up

-- For each Stage and each Target it has promoted to: the Stage's latest
-- Promotion to the Target, and its latest successful one, whose Freight the
-- Target runs. Fleet views need both for every pair at once, and finding
-- them in the Promotion history costs one search per pair, over a history
-- that only grows; here each pair is one row, however many Promotions it
-- has had. The rows are derived from Promotion rows and written in the same
-- statements that write those, so they never trail them. Stage and Target
-- are recorded by name, as Promotion rows record them, so the history of a
-- name survives the Stage or Target being recreated.
CREATE TABLE target_stage_promotions (
    project_name TEXT NOT NULL,
    target_name TEXT NOT NULL CHECK (target_name <> ''),
    stage_name TEXT NOT NULL,
    -- The latest Promotion by creation time, then id, whatever its phase.
    -- Promotion rows are never deleted, so neither reference can dangle.
    latest_promotion_id TEXT NOT NULL REFERENCES promotions (id),
    -- The latest Promotion's phase, copied so that counting pairs by state
    -- needs no join.
    latest_phase TEXT NOT NULL,
    latest_created_at TIMESTAMPTZ NOT NULL,
    -- The latest successful Promotion by finish time, then id. NULL until a
    -- Promotion from the Stage to the Target has succeeded.
    succeeded_promotion_id TEXT REFERENCES promotions (id),
    succeeded_finished_at TIMESTAMPTZ,
    PRIMARY KEY (project_name, target_name, stage_name),
    CHECK ((succeeded_promotion_id IS NULL) = (succeeded_finished_at IS NULL))
);

-- Serves a Stage's Targets with their state.
CREATE INDEX target_stage_promotions_stage_idx
    ON target_stage_promotions (project_name, stage_name);

-- Records the pairs the Promotion history already holds.
INSERT INTO target_stage_promotions (
    project_name,
    target_name,
    stage_name,
    latest_promotion_id,
    latest_phase,
    latest_created_at,
    succeeded_promotion_id,
    succeeded_finished_at
)
SELECT
    latest.project_name,
    latest.target_name,
    latest.stage_name,
    latest.id,
    latest.phase,
    latest.created_at,
    succeeded.id,
    succeeded.finished_at
FROM (
    SELECT DISTINCT ON (project_name, target_name, stage_name)
        project_name, target_name, stage_name, id, phase, created_at
    FROM promotions
    WHERE target_name IS NOT NULL
    ORDER BY project_name, target_name, stage_name, created_at DESC, id DESC
) AS latest
LEFT JOIN (
    SELECT DISTINCT ON (project_name, target_name, stage_name)
        project_name, target_name, stage_name, id, finished_at
    FROM promotions
    WHERE target_name IS NOT NULL
        AND phase = 'Succeeded'
        AND finished_at IS NOT NULL
    ORDER BY project_name, target_name, stage_name, finished_at DESC, id DESC
) AS succeeded USING (project_name, target_name, stage_name);

-- +goose Down
DROP TABLE target_stage_promotions;
