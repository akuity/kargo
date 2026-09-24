-- +goose Up
-- Targets and PromotionRequests are authored in the database rather than
-- mirrored from Kubernetes, so they carry updated_at rather than synced_at.
CREATE TABLE targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(labels) = 'object'),
    params JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(params) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, name)
);

CREATE TABLE promotion_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- seq is the creation order. Sort and compare by it, never by name.
    seq BIGINT GENERATED ALWAYS AS IDENTITY NOT NULL UNIQUE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    stage_id TEXT NOT NULL REFERENCES stages(id) ON DELETE CASCADE,
    freight_id TEXT NOT NULL REFERENCES freight(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_by TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT 'Pending'
        CHECK (phase IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Errored')),
    message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    UNIQUE (project_id, name)
);

CREATE INDEX promotion_requests_stage ON promotion_requests (stage_id, seq);
CREATE INDEX promotion_requests_open ON promotion_requests (id)
WHERE phase IN ('Pending', 'Running');

-- One row per Target the request fans out to: the snapshot taken when the
-- request was created, plus that Target's outcome. The ordinal preserves the
-- order in which the Targets were resolved.
CREATE TABLE promotion_request_targets (
    promotion_request_id UUID NOT NULL REFERENCES promotion_requests(id) ON DELETE CASCADE,
    target_id UUID NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    promotion TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (promotion_request_id, target_id)
);

CREATE INDEX promotion_request_targets_target ON promotion_request_targets (target_id);

-- +goose Down
DROP TABLE IF EXISTS promotion_request_targets;
DROP TABLE IF EXISTS promotion_requests;
DROP TABLE IF EXISTS targets;
