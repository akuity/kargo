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
    -- The TargetSet that manages this Target's lifecycle, by name within the
    -- Project. Every Target has one. A TargetSet cannot be renamed, so its
    -- name identifies it for its lifetime; TargetSets are not rows yet, so
    -- nothing references them, and a TargetSet removes its own Targets
    -- before it goes away.
    target_set_name TEXT NOT NULL CHECK (target_set_name <> ''),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT targets_project_name_name_key UNIQUE (project_name, name)
);
-- Serves a TargetSet finding its own Targets.
CREATE INDEX targets_target_set_name_idx ON targets (project_name, target_set_name);

-- +goose Down
DROP TABLE targets;
DROP FUNCTION jsonb_is_string_map(jsonb);
