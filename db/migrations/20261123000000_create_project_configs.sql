-- +goose Up

-- A ProjectConfig is authored in the database and belongs to the Project of
-- the same name in Kubernetes. There is deliberately no foreign key to the
-- mirrored projects table: the project wizard creates a Project and its
-- ProjectConfig in one request, and a reference would fail that request
-- until the mirror caught up. Nothing in the database removes a
-- ProjectConfig when its Project is deleted; that is the job of whatever
-- handles the Project's deletion.
--
-- project_uid is the UID of the Kubernetes Project the row was created for.
-- It tells a deleted Project's leftover row from the config of a Project
-- recreated under the same name, for cleanup and for filtering reads.
--
-- updated_at doubles as the row's version, as it does for targets.
CREATE TABLE project_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_name TEXT NOT NULL,
    project_uid TEXT NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_is_string_map(labels)),
    annotations JSONB NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_is_string_map(annotations)),
    spec JSONB NOT NULL CHECK (jsonb_typeof(spec) = 'object'),
    generation BIGINT NOT NULL DEFAULT 1,
    refresh_token TEXT,
    status_conditions JSONB,
    status_observed_generation BIGINT,
    status_last_handled_refresh TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT project_configs_project_name_key UNIQUE (project_name)
);

-- The queryable half of a ProjectConfig's status. The path is the lookup key
-- for an incoming webhook, so at most one ProjectConfig may own a path.
CREATE TABLE project_config_webhook_receivers (
    project_config_id UUID NOT NULL REFERENCES project_configs (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    url TEXT NOT NULL,
    PRIMARY KEY (project_config_id, name),
    CONSTRAINT project_config_webhook_receivers_path_key UNIQUE (path)
);

-- +goose Down
DROP TABLE project_config_webhook_receivers;
DROP TABLE project_configs;
