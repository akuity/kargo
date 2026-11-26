-- A ProjectConfig is addressed by Project name. A row belongs to an earlier
-- Project of the same name when the mirrored projects table holds a different
-- UID for its name; queries that cannot be told the current UID hide such rows
-- until the row is adopted by the current Project. A Project that is not
-- mirrored yet hides nothing.

-- Reads by name alone; the store applies the caller's Project UID.
-- name: GetProjectConfig :one
SELECT * FROM project_configs
WHERE project_name = sqlc.arg(project_name);

-- name: ListProjectConfigs :many
SELECT pc.* FROM project_configs AS pc
LEFT JOIN projects AS p ON p.name = pc.project_name
WHERE (p.id IS NULL OR p.id = pc.project_uid)
ORDER BY pc.project_name;

-- name: InsertProjectConfig :one
INSERT INTO project_configs (
    project_name,
    project_uid,
    labels,
    annotations,
    spec
)
VALUES (
    sqlc.arg(project_name)::text,
    sqlc.arg(project_uid)::text,
    sqlc.arg(labels)::jsonb,
    sqlc.arg(annotations)::jsonb,
    sqlc.arg(spec)::jsonb
)
RETURNING *;

-- Imports a config from its Kubernetes resource along with the conditions
-- of its last reconcile. The observed generation is left NULL so that the
-- reconciler verifies the config again. An existing row is never
-- overwritten: no row returned means one exists for the Project.
-- name: ImportProjectConfig :one
INSERT INTO project_configs (
    project_name,
    project_uid,
    labels,
    annotations,
    spec,
    status_conditions
)
VALUES (
    sqlc.arg(project_name)::text,
    sqlc.arg(project_uid)::text,
    sqlc.arg(labels)::jsonb,
    sqlc.arg(annotations)::jsonb,
    sqlc.arg(spec)::jsonb,
    sqlc.narg(status_conditions)::jsonb
)
ON CONFLICT (project_name) DO NOTHING
RETURNING *;

-- Returns the deleted row because the caller needs it to report the event.
-- name: DeleteProjectConfigByID :one
DELETE FROM project_configs
WHERE id = sqlc.arg(id)
RETURNING *;

-- generation counts changes to spec only, and jsonb equality ignores key
-- order, so rewriting an identical spec does not bump it. updated_at doubles
-- as the row's version, so it must grow with every write to a row even across
-- a clock step or two writes in one microsecond. A non-null id or updated_at
-- is a further precondition; a row that fails one is left alone and nothing
-- is returned.
-- name: UpdateProjectConfigSpec :one
UPDATE project_configs SET
    labels = sqlc.arg(labels)::jsonb,
    annotations = sqlc.arg(annotations)::jsonb,
    spec = sqlc.arg(spec)::jsonb,
    generation = generation + CASE
        WHEN spec IS DISTINCT FROM sqlc.arg(spec)::jsonb THEN 1
        ELSE 0
    END,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND (sqlc.narg(id)::uuid IS NULL OR id = sqlc.narg(id))
    AND (
        sqlc.narg(updated_at)::timestamptz IS NULL
        OR updated_at = sqlc.narg(updated_at)
    )
RETURNING *;

-- name: SetProjectConfigRefreshToken :one
UPDATE project_configs SET
    refresh_token = sqlc.arg(refresh_token)::text,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
RETURNING *;

-- Rebinds a row to the Project of the same name that now holds the name,
-- keeping everything else. It matches only the row read just before, and only
-- while it belongs to another Project UID.
-- name: AdoptProjectConfig :one
UPDATE project_configs SET
    project_uid = sqlc.arg(project_uid)::text,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND id = sqlc.arg(id)
    AND project_uid <> sqlc.arg(project_uid)::text
RETURNING *;

-- A status write is a reconcile of a particular row, generation and refresh
-- token. If the row has been replaced, or either has moved on since the
-- reconcile read them, nothing is written and nothing is returned.
-- name: UpdateProjectConfigStatus :one
UPDATE project_configs SET
    status_conditions = sqlc.arg(status_conditions)::jsonb,
    status_observed_generation = sqlc.arg(status_observed_generation)::bigint,
    status_last_handled_refresh = sqlc.narg(status_last_handled_refresh)::text,
    updated_at = GREATEST(clock_timestamp(), updated_at + interval '1 microsecond')
WHERE project_name = sqlc.arg(project_name)
    AND id = sqlc.arg(id)
    AND generation = sqlc.arg(observed_generation)::bigint
    AND refresh_token IS NOT DISTINCT FROM sqlc.narg(observed_refresh_token)::text
RETURNING *;

-- name: DeleteProjectConfigWebhookReceivers :exec
DELETE FROM project_config_webhook_receivers
WHERE project_config_id = sqlc.arg(project_config_id);

-- No row returned means this receiver was skipped: its path already belongs
-- to another ProjectConfig, or this ProjectConfig already has a receiver of
-- that name.
-- name: InsertProjectConfigWebhookReceiver :one
INSERT INTO project_config_webhook_receivers (
    project_config_id,
    name,
    path,
    url
)
VALUES (
    sqlc.arg(project_config_id)::uuid,
    sqlc.arg(name)::text,
    sqlc.arg(path)::text,
    sqlc.arg(url)::text
)
ON CONFLICT DO NOTHING
RETURNING name;

-- name: ListProjectConfigWebhookReceivers :many
SELECT * FROM project_config_webhook_receivers
WHERE project_config_id = sqlc.arg(project_config_id)
ORDER BY name;

-- name: ListProjectConfigWebhookReceiversByIDs :many
SELECT * FROM project_config_webhook_receivers
WHERE project_config_id = ANY(sqlc.arg(project_config_ids)::uuid[])
ORDER BY project_config_id, name;

-- name: GetProjectConfigByWebhookReceiverPath :one
SELECT pc.* FROM project_configs AS pc
JOIN project_config_webhook_receivers AS r ON r.project_config_id = pc.id
LEFT JOIN projects AS p ON p.name = pc.project_name
WHERE r.path = sqlc.arg(path)
    AND (p.id IS NULL OR p.id = pc.project_uid);
