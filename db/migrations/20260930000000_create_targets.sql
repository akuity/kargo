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
-- mirrored Project row and goes with it: when a Project is deleted, or
-- recreated under the same name with a new identity, its Targets are gone.
CREATE TABLE targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_is_string_map(labels)),
    params JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(params) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT targets_project_id_name_key UNIQUE (project_id, name)
);

-- +goose Down
DROP TABLE targets;
DROP FUNCTION jsonb_is_string_map(jsonb);
