-- +goose Up

-- A rollout promotes to Targets of its own Project and no other. Until now
-- only the writer saw to that: a snapshot row referenced a rollout and a
-- Target by their ids, and nothing held the two to one Project, so a Target
-- of another Project could be recorded with a rollout and then read back,
-- by name, as one of that rollout's.
--
-- Each snapshot row now carries the Project's name, and both of its
-- references include it: the row's rollout and its Target must each be the
-- one of that id in that Project. A Target of another Project fails the
-- reference, in the statement that records it, whoever the writer is. The
-- reference also keeps holding afterwards: neither a rollout nor a Target
-- can leave its Project while a snapshot row refers to it.

-- A foreign key must reference a unique key. The id alone is one in both
-- tables, so these add nothing but the references' targets.
ALTER TABLE targets
    ADD CONSTRAINT targets_id_project_name_key UNIQUE (id, project_name);
ALTER TABLE promotion_rollouts
    ADD CONSTRAINT promotion_rollouts_id_project_name_key UNIQUE (id, project_name);

ALTER TABLE promotion_rollout_targets ADD COLUMN project_name TEXT;
UPDATE promotion_rollout_targets SET project_name = promotion_rollouts.project_name
FROM promotion_rollouts
WHERE promotion_rollouts.id = promotion_rollout_targets.promotion_rollout_id;

-- Rows already here that pair a rollout with a Target of another Project
-- would be refused under this rule, and go now.
DELETE FROM promotion_rollout_targets
USING targets
WHERE targets.id = promotion_rollout_targets.target_id
    AND targets.project_name <> promotion_rollout_targets.project_name;

-- The primary key still serves a rollout's snapshot and the cascade from
-- the rollout, and promotion_rollout_targets_target_idx the cascade from
-- the Target.
ALTER TABLE promotion_rollout_targets
    ALTER COLUMN project_name SET NOT NULL,
    DROP CONSTRAINT promotion_rollout_targets_promotion_rollout_id_fkey,
    ADD CONSTRAINT promotion_rollout_targets_promotion_rollout_fkey
        FOREIGN KEY (promotion_rollout_id, project_name)
        REFERENCES promotion_rollouts (id, project_name) ON DELETE CASCADE,
    DROP CONSTRAINT promotion_rollout_targets_target_id_fkey,
    ADD CONSTRAINT promotion_rollout_targets_target_fkey
        FOREIGN KEY (target_id, project_name)
        REFERENCES targets (id, project_name) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE promotion_rollout_targets
    DROP CONSTRAINT promotion_rollout_targets_target_fkey,
    ADD CONSTRAINT promotion_rollout_targets_target_id_fkey
        FOREIGN KEY (target_id) REFERENCES targets (id) ON DELETE CASCADE,
    DROP CONSTRAINT promotion_rollout_targets_promotion_rollout_fkey,
    ADD CONSTRAINT promotion_rollout_targets_promotion_rollout_id_fkey
        FOREIGN KEY (promotion_rollout_id)
        REFERENCES promotion_rollouts (id) ON DELETE CASCADE,
    DROP COLUMN project_name;
ALTER TABLE promotion_rollouts DROP CONSTRAINT promotion_rollouts_id_project_name_key;
ALTER TABLE targets DROP CONSTRAINT targets_id_project_name_key;
