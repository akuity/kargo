-- +goose Up

-- A PromotionRollout promotes one piece of Freight from one Stage to the
-- Targets the Stage governed when the rollout was created. Like Targets,
-- rollouts are authored in the database and belong to the Project of that
-- name; no reference to the mirrored Project row is kept, and nothing here
-- removes a Project's rollouts when the Project is deleted.
--
-- The name is <stage>.<ulid>.<short-hash>. Among one Stage's rollouts,
-- number order is creation order; name order nearly is, but the ULID is
-- minted before the number is assigned, so two rollouts created at once can
-- be numbered in the opposite order to their names. stage_name and
-- freight_name are fixed for the rollout's lifetime.
CREATE TABLE promotion_rollouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    -- The rollout's position among its Stage's rollouts, from 1.
    number BIGINT NOT NULL CHECK (number > 0),
    freight_name TEXT NOT NULL,
    -- The Stage's update strategy when the rollout was created, as the Stage
    -- gives it: a number of Targets, or a percentage of the rollout's
    -- Targets such as '10%', which the reconciler resolves once it knows
    -- them. NULL max_concurrent is no limit; NULL max_failures is none
    -- tolerated.
    max_concurrent TEXT CHECK (max_concurrent ~ '^[1-9][0-9]*%?$'),
    max_failures TEXT CHECK (max_failures ~ '^(0|[1-9][0-9]*)%?$'),
    created_by TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    -- When the reconciler resolved the Stage's selectors into the rollout's
    -- Targets (promotion_rollout_targets). NULL until it has: a rollout
    -- with no Target rows is one that has not been resolved yet, or one
    -- that resolved to nothing, and this tells the two apart.
    targets_resolved_at TIMESTAMPTZ,
    -- The status_* columns are the rollout's status, written only by the
    -- reconciler. The API presents status_phase, status_reason and
    -- status_message as the rollout's Ready condition as well. What has
    -- become of each Target is not recorded here: the Promotions that
    -- promote to them say so themselves.
    status_phase TEXT NOT NULL DEFAULT 'Pending' CHECK (
        status_phase IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Errored', 'Aborted')
    ),
    status_reason TEXT NOT NULL DEFAULT '',
    status_message TEXT NOT NULL DEFAULT '',
    status_started_at TIMESTAMPTZ,
    status_finished_at TIMESTAMPTZ,
    CONSTRAINT promotion_rollouts_project_name_name_key UNIQUE (project_name, name),
    CONSTRAINT promotion_rollouts_stage_number_key UNIQUE (project_name, stage_name, number)
);

-- A Project's rollouts are listed newest first, a page at a time; a Stage's
-- are served from the stage/number unique index above, in either direction.
CREATE INDEX promotion_rollouts_created_idx
    ON promotion_rollouts (project_name, created_at, id);

-- A Stage's rollouts of one piece of Freight, newest first.
CREATE INDEX promotion_rollouts_freight_idx
    ON promotion_rollouts (project_name, stage_name, freight_name, number);

-- The rollouts still in progress, by Stage: the reconciler's resync relists
-- them, and a Stage's current rollout is one of them. However many rollouts
-- a Stage has had, few are ever open at once. The predicate must match
-- the queries' word for word.
CREATE INDEX promotion_rollouts_open_idx
    ON promotion_rollouts (project_name, stage_name, number)
    WHERE status_phase IN ('Pending', 'Running');

-- A Stage promotes through one rollout at a time. StartPromotionRollout
-- moves a rollout to Running only when it is the next of its Stage's to
-- run, and this index is the guarantee behind it: however the claims race,
-- two rollouts of one Stage cannot both be Running. The Stage's own status
-- records nothing about which rollout is in progress; this is where that is
-- decided.
CREATE UNIQUE INDEX promotion_rollouts_running_key
    ON promotion_rollouts (project_name, stage_name)
    WHERE status_phase = 'Running';

-- The Targets a rollout promotes to: the snapshot the reconciler takes, once,
-- when it first sees the rollout, one row per Target and nothing else.
-- Targets are database-native, so the snapshot references their rows.
-- Deleting a Target removes it from every rollout's snapshot; deleting a
-- rollout removes its snapshot.
CREATE TABLE promotion_rollout_targets (
    promotion_rollout_id UUID NOT NULL REFERENCES promotion_rollouts (id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES targets (id) ON DELETE CASCADE,
    PRIMARY KEY (promotion_rollout_id, target_id)
);

CREATE INDEX promotion_rollout_targets_target_idx ON promotion_rollout_targets (target_id);

-- The PromotionRollout a mirrored Promotion was created by, as one part of
-- its fan-out to Targets. NULL for a Promotion created on its own. The
-- mirror records it from the Promotion's kargo.akuity.io/promotion-rollout
-- annotation, which carries the rollout's id; nothing that creates
-- Promotions writes it here. Once recorded it is kept, like the rest of the
-- Promotion's record. A rollout is never deleted, and while Promotion rows
-- refer to it, it cannot be: the history they record would lose its rollout.
ALTER TABLE promotions
    ADD COLUMN promotion_rollout_id UUID REFERENCES promotion_rollouts (id);

-- Serves a rollout's Promotions.
CREATE INDEX promotions_promotion_rollout_idx
    ON promotions (promotion_rollout_id)
    WHERE promotion_rollout_id IS NOT NULL;

-- +goose Down
ALTER TABLE promotions DROP COLUMN promotion_rollout_id;
DROP TABLE promotion_rollout_targets;
DROP TABLE promotion_rollouts;
