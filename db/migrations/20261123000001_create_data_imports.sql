-- +goose Up

-- Records each one-time import of data into the database that has completed,
-- by name. An import whose name is here never runs again, so that, for
-- instance, a ProjectConfig deleted after the import of ProjectConfig
-- resources is not brought back from its resource.
CREATE TABLE data_imports (
    name TEXT PRIMARY KEY,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE data_imports;
