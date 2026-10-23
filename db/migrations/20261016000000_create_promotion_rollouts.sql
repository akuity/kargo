-- +goose Up

-- A PromotionRollout promotes one piece of Freight from one Stage to the
-- Targets the Stage governed when the rollout was created. Like Targets,
-- rollouts are authored in the database and belong to the Project of that
-- name; no reference to the mirrored Project row is kept, and nothing here
-- removes a Project's rollouts when the Project is deleted.
--
-- The name is <stage>.<ulid>.<short-hash>, so among one Stage's rollouts
-- name order is creation order, as is number order. stage_name and
-- freight_name are fixed for the rollout's lifetime.
CREATE TABLE promotion_rollouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    -- The rollout's position among its Stage's rollouts, from 1.
    number BIGINT NOT NULL CHECK (number > 0),
    freight_name TEXT NOT NULL,
    max_concurrent BIGINT NOT NULL DEFAULT 0 CHECK (max_concurrent >= 0),
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

-- Project-wide lists are in creation order; a Stage's rollouts are served
-- from the stage/number unique index above.
CREATE INDEX promotion_rollouts_created_idx
    ON promotion_rollouts (project_name, created_at, number);

-- The rollouts still in progress, which the reconciler relists on every
-- resync. The predicate must match ListOpenPromotionRollouts word for word.
CREATE INDEX promotion_rollouts_open_idx
    ON promotion_rollouts (project_name, name)
    WHERE status_phase IN ('Pending', 'Running');

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

-- +goose Down
DROP TABLE promotion_rollout_targets;
DROP TABLE promotion_rollouts;
