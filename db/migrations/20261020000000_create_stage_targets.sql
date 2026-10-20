-- +goose Up

-- Which Stage governs which Target: one row per Stage whose target selectors
-- a Target's labels match. The rows are derived, never authored: the
-- management controller recomputes a Project's rows from its Stages' mirrored
-- selectors and its Targets' labels whenever either changes, and a row goes
-- with its Stage or its Target.
CREATE TABLE stage_targets (
    stage_id TEXT NOT NULL REFERENCES stages (id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES targets (id) ON DELETE CASCADE,
    PRIMARY KEY (stage_id, target_id)
);
-- Serves finding the Stages that govern a Target.
CREATE INDEX stage_targets_target_id_idx ON stage_targets (target_id);

-- +goose Down
DROP TABLE stage_targets;
