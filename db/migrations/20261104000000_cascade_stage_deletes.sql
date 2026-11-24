-- +goose Up

-- A Stage's rows go when the Stage does, as its Promotions do in Kubernetes,
-- where the Stage owns them and they are collected with it. Promotion rows
-- used to outlive their Stage as a history, and PromotionRollouts, which
-- exist only here, were never deleted at all. Both now reference the Stage's
-- mirrored row and are deleted with it, and their own children follow: a
-- Promotion's Freight, a rollout's Target snapshot and its child Promotions.
-- A Project's deletion deletes its Stages in Kubernetes, so it now clears
-- the Project's Promotions and rollouts too, which nothing did before.
--
-- A Promotion's row is still kept past the Promotion's own deletion:
-- Kubernetes retains a bounded number per Stage, and the database keeps
-- every one for as long as the Stage lives. A Stage deleted and recreated
-- under the same name is a new row with a new id, so when the mirror
-- replaces the old row the old Stage's history goes with it, and the new
-- Stage's rollouts are numbered from 1.
--
-- The Stage is referenced by its id, the Kubernetes UID, beside the name the
-- rows already carry. The name remains how rows are read and listed; the id
-- ties each to one Stage's lifetime. A writer resolves the id from the name
-- through the stages table in the same statement, so a Stage the mirror has
-- not recorded yet refuses the write, as the rollout number already did,
-- until the mirror catches up.
--
-- Rows already here whose Stage has no mirrored row would have gone with it
-- under this rule, and go now.

ALTER TABLE promotions
    ADD COLUMN stage_id TEXT REFERENCES stages (id) ON DELETE CASCADE;
UPDATE promotions SET stage_id = stages.id
FROM stages
WHERE stages.project_name = promotions.project_name
    AND stages.name = promotions.stage_name;
DELETE FROM promotions WHERE stage_id IS NULL;
ALTER TABLE promotions ALTER COLUMN stage_id SET NOT NULL;
-- Serves the cascade, which would otherwise scan the whole history.
CREATE INDEX promotions_stage_id_idx ON promotions (stage_id);

-- A Promotion goes with its rollout as it goes with its Stage. Today only a
-- Stage's deletion deletes a rollout, which deletes the Promotion either
-- way; the cascade says what a rollout's deletion means should another
-- way come.
ALTER TABLE promotions
    DROP CONSTRAINT promotions_promotion_rollout_id_fkey,
    ADD CONSTRAINT promotions_promotion_rollout_id_fkey
        FOREIGN KEY (promotion_rollout_id)
        REFERENCES promotion_rollouts (id) ON DELETE CASCADE;

ALTER TABLE promotion_rollouts
    ADD COLUMN stage_id TEXT REFERENCES stages (id) ON DELETE CASCADE;
UPDATE promotion_rollouts SET stage_id = stages.id
FROM stages
WHERE stages.project_name = promotion_rollouts.project_name
    AND stages.name = promotion_rollouts.stage_name;
DELETE FROM promotion_rollouts WHERE stage_id IS NULL;
ALTER TABLE promotion_rollouts ALTER COLUMN stage_id SET NOT NULL;
CREATE INDEX promotion_rollouts_stage_id_idx ON promotion_rollouts (stage_id);

-- +goose Down
DROP INDEX promotion_rollouts_stage_id_idx;
ALTER TABLE promotion_rollouts DROP COLUMN stage_id;
ALTER TABLE promotions
    DROP CONSTRAINT promotions_promotion_rollout_id_fkey,
    ADD CONSTRAINT promotions_promotion_rollout_id_fkey
        FOREIGN KEY (promotion_rollout_id) REFERENCES promotion_rollouts (id);
DROP INDEX promotions_stage_id_idx;
ALTER TABLE promotions DROP COLUMN stage_id;
