-- +goose Up

-- Stages are mirrored from Kubernetes, which remains their source of truth:
-- a row records a Stage's identity and is refreshed by the management
-- controller as the Stage changes. Like a Target, a Stage names its Project
-- rather than referencing the mirrored Project row, since the name is the
-- Project's identity for its lifetime. A Project's deletion deletes its
-- Stages, and the mirror removes their rows as Kubernetes confirms each one
-- gone.
CREATE TABLE stages (
    id TEXT PRIMARY KEY,
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT stages_project_name_name_key UNIQUE (project_name, name)
);

-- Promotions are mirrored from Kubernetes too, but unlike other mirrored
-- kinds their rows are never deleted: Kubernetes keeps only a bounded number
-- of recent Promotions per Stage, and this table keeps every one of them as
-- a history. A row is written when a Promotion first appears and refreshed
-- as its phase changes, until it is deleted from Kubernetes and the row
-- stays behind as its record. A Promotion names its Project and Stage rather
-- than referencing their mirrored rows, since both may be gone while the
-- record remains, and likewise names its Target, when it promotes to one,
-- rather than referencing the Target's row. Names are not unique here: a
-- Promotion deleted and recreated under the same name is a different
-- Promotion with a row of its own.
CREATE TABLE promotions (
    id TEXT PRIMARY KEY,
    project_name TEXT NOT NULL,
    name TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    freight_name TEXT NOT NULL,
    -- NULL for a Promotion that promotes to no Target: one of a classic
    -- Stage, which governs none.
    target_name TEXT CHECK (target_name <> ''),
    phase TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    synced_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Serves a Stage's promotion history, newest first.
CREATE INDEX promotions_stage_created_at_idx
    ON promotions (project_name, stage_name, created_at DESC);
-- Serves a Target's promotion history, newest first.
CREATE INDEX promotions_target_created_at_idx
    ON promotions (project_name, target_name, created_at DESC)
    WHERE target_name IS NOT NULL;
-- Serves lookups by the name Kubernetes knew the Promotion by.
CREATE INDEX promotions_project_name_name_idx ON promotions (project_name, name);

-- +goose Down
DROP TABLE promotions;
DROP TABLE stages;
