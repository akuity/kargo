-- +goose Up

-- A label map is a JSON object whose values are all strings. CHECK
-- constraints cannot contain subqueries, so the test lives in a function.
-- +goose StatementBegin
CREATE FUNCTION jsonb_is_string_map(value jsonb) RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS $$
    SELECT jsonb_typeof(value) = 'object'
        AND NOT EXISTS (
            SELECT 1 FROM jsonb_each(value) AS entry
            WHERE jsonb_typeof(entry.value) <> 'string'
        )
$$;
-- +goose StatementEnd

-- Targets are authored in the database rather than mirrored from Kubernetes,
-- so they carry updated_at rather than synced_at. A Target belongs to the
-- Project of that name in Kubernetes. The name is the Project's identity for
-- its lifetime, since Kubernetes objects cannot be renamed, so no reference
-- to a mirrored Project row is needed and none is kept: a write never has to
-- wait for a mirror to catch up. Nothing in the database removes a Project's
-- Targets when the Project is deleted; that is the job of whatever handles
-- the Project's deletion.
CREATE TABLE targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_is_string_map(labels)),
    params JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(params) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT targets_project_name_name_key UNIQUE (project_name, name)
);

-- A Stage as the database knows it. Stages live in Kubernetes and are not
-- mirrored yet: a row is created the first time a PromotionRequest is made
-- for the Stage, and holds only what the database needs of it, which for now
-- is the counter that numbers its PromotionRequests.
CREATE TABLE stages (
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    -- The number given to the Stage's most recent PromotionRequest.
    last_promotion_request_number BIGINT NOT NULL DEFAULT 0
        CHECK (last_promotion_request_number >= 0),
    PRIMARY KEY (project_name, name)
);

-- A PromotionRequest is the intent to promote a piece of Freight, through a
-- Stage, to the Targets the Stage governed when the request was made. Like
-- Targets, PromotionRequests are authored in the database. The Stage and
-- Freight are named, not referenced: both live in Kubernetes, and a request
-- is history that must outlive them. A request's phase is that of the
-- PromotionRequest API type.
CREATE TABLE promotion_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    -- number counts the Stage's PromotionRequests from 1, without gaps, like
    -- the runs of a workflow. Within a Stage it is the creation order: sort
    -- and compare by it, never by name.
    number BIGINT NOT NULL CHECK (number > 0),
    freight_name TEXT NOT NULL,
    update_strategy JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(update_strategy) = 'object'),
    created_by TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT 'Pending' CHECK (
        phase IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Errored', 'Aborted')
    ),
    message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    CONSTRAINT promotion_requests_project_name_name_key UNIQUE (project_name, name),
    CONSTRAINT promotion_requests_stage_number_key
        UNIQUE (project_name, stage_name, number)
);

CREATE INDEX promotion_requests_open_idx
    ON promotion_requests (project_name, stage_name, number)
    WHERE phase IN ('Pending', 'Running');

-- One row per Target a PromotionRequest fans out to: the snapshot taken when
-- the request was created, plus that Target's outcome. The Target is named,
-- not referenced: a Target deleted mid-flight must neither take the request with
-- it nor be held back by it. Join to targets on (project_name, name) to
-- reach the Target itself, for as long as it exists.
CREATE TABLE promotion_request_targets (
    promotion_request_id UUID NOT NULL
        REFERENCES promotion_requests(id) ON DELETE CASCADE,
    target_name TEXT NOT NULL,
    -- promotion names the child Promotion promoting to the Target, once one
    -- exists, and phase is that Promotion's phase.
    promotion TEXT NOT NULL DEFAULT '',
    phase TEXT NOT NULL DEFAULT '' CHECK (
        phase IN ('', 'Pending', 'Running', 'Succeeded', 'Failed', 'Errored', 'Aborted')
    ),
    PRIMARY KEY (promotion_request_id, target_name)
);

CREATE INDEX promotion_request_targets_target_name_idx
    ON promotion_request_targets (target_name);

-- +goose Down
DROP TABLE promotion_request_targets;
DROP TABLE promotion_requests;
DROP TABLE stages;
DROP TABLE targets;
DROP FUNCTION jsonb_is_string_map(jsonb);
