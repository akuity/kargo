-- +goose Up
CREATE TABLE warehouses (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    UNIQUE (project_id, id)
);

CREATE UNIQUE INDEX warehouses_active_name ON warehouses (project_id, name)
WHERE deleted_at IS NULL;

CREATE TABLE freight (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    warehouse_id TEXT NOT NULL,
    name TEXT NOT NULL,
    alias TEXT NOT NULL,
    discovered_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (project_id, warehouse_id)
        REFERENCES warehouses(project_id, id) ON DELETE CASCADE
);

CREATE INDEX freight_warehouse ON freight (project_id, warehouse_id);
CREATE UNIQUE INDEX freight_active_name ON freight (project_id, name)
WHERE deleted_at IS NULL;

-- Ordinals preserve the order within each Kubernetes artifact list.
CREATE TABLE freight_commits (
    freight_id TEXT NOT NULL REFERENCES freight(id) ON DELETE CASCADE,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    repo_url TEXT NOT NULL,
    commit_id TEXT NOT NULL,
    branch TEXT NOT NULL,
    tag TEXT NOT NULL,
    message TEXT NOT NULL,
    author TEXT NOT NULL,
    committer TEXT NOT NULL,
    subscription_name TEXT NOT NULL,
    PRIMARY KEY (freight_id, ordinal)
);

CREATE TABLE freight_images (
    freight_id TEXT NOT NULL REFERENCES freight(id) ON DELETE CASCADE,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    repo_url TEXT NOT NULL,
    tag TEXT NOT NULL,
    digest TEXT NOT NULL,
    subscription_name TEXT NOT NULL,
    annotations JSONB NOT NULL CHECK (jsonb_typeof(annotations) = 'object'),
    PRIMARY KEY (freight_id, ordinal)
);

CREATE TABLE freight_charts (
    freight_id TEXT NOT NULL REFERENCES freight(id) ON DELETE CASCADE,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    repo_url TEXT NOT NULL,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    subscription_name TEXT NOT NULL,
    PRIMARY KEY (freight_id, ordinal)
);

CREATE TABLE freight_artifacts (
    freight_id TEXT NOT NULL REFERENCES freight(id) ON DELETE CASCADE,
    ordinal BIGINT NOT NULL CHECK (ordinal >= 0),
    artifact_type TEXT NOT NULL,
    subscription_name TEXT NOT NULL,
    version TEXT NOT NULL,
    metadata JSONB CHECK (jsonb_typeof(metadata) = 'object'),
    PRIMARY KEY (freight_id, ordinal)
);

-- +goose Down
DROP TABLE IF EXISTS freight_artifacts;
DROP TABLE IF EXISTS freight_charts;
DROP TABLE IF EXISTS freight_images;
DROP TABLE IF EXISTS freight_commits;

DROP TABLE IF EXISTS freight;
DROP TABLE IF EXISTS warehouses;
