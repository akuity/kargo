-- name: UpsertFreightCommit :exec
INSERT INTO freight_commits (freight_id, ordinal, repo_url, commit_id, branch, tag, message, author, committer, subscription_name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (freight_id, ordinal) DO UPDATE SET
    repo_url = EXCLUDED.repo_url,
    commit_id = EXCLUDED.commit_id,
    branch = EXCLUDED.branch,
    tag = EXCLUDED.tag,
    message = EXCLUDED.message,
    author = EXCLUDED.author,
    committer = EXCLUDED.committer,
    subscription_name = EXCLUDED.subscription_name;

-- name: DeleteFreightCommits :exec
DELETE FROM freight_commits WHERE freight_id = $1;

-- name: ListFreightCommits :many
SELECT a.freight_id, a.ordinal, a.repo_url, a.commit_id, a.branch, a.tag, a.message, a.author, a.committer, a.subscription_name FROM freight_commits a
JOIN freight f ON f.id = a.freight_id
WHERE f.deleted_at IS NULL
ORDER BY a.freight_id, a.ordinal;

-- name: UpsertFreightImage :exec
INSERT INTO freight_images (freight_id, ordinal, repo_url, tag, digest, subscription_name, annotations)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (freight_id, ordinal) DO UPDATE SET
    repo_url = EXCLUDED.repo_url,
    tag = EXCLUDED.tag,
    digest = EXCLUDED.digest,
    subscription_name = EXCLUDED.subscription_name,
    annotations = EXCLUDED.annotations;

-- name: DeleteFreightImages :exec
DELETE FROM freight_images WHERE freight_id = $1;

-- name: ListFreightImages :many
SELECT a.freight_id, a.ordinal, a.repo_url, a.tag, a.digest, a.subscription_name, a.annotations FROM freight_images a
JOIN freight f ON f.id = a.freight_id
WHERE f.deleted_at IS NULL
ORDER BY a.freight_id, a.ordinal;

-- name: UpsertFreightChart :exec
INSERT INTO freight_charts (freight_id, ordinal, repo_url, name, version, subscription_name)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (freight_id, ordinal) DO UPDATE SET
    repo_url = EXCLUDED.repo_url,
    name = EXCLUDED.name,
    version = EXCLUDED.version,
    subscription_name = EXCLUDED.subscription_name;

-- name: DeleteFreightCharts :exec
DELETE FROM freight_charts WHERE freight_id = $1;

-- name: ListFreightCharts :many
SELECT a.freight_id, a.ordinal, a.repo_url, a.name, a.version, a.subscription_name FROM freight_charts a
JOIN freight f ON f.id = a.freight_id
WHERE f.deleted_at IS NULL
ORDER BY a.freight_id, a.ordinal;

-- name: UpsertFreightArtifact :exec
INSERT INTO freight_artifacts (freight_id, ordinal, artifact_type, subscription_name, version, metadata)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (freight_id, ordinal) DO UPDATE SET
    artifact_type = EXCLUDED.artifact_type,
    subscription_name = EXCLUDED.subscription_name,
    version = EXCLUDED.version,
    metadata = EXCLUDED.metadata;

-- name: DeleteFreightArtifacts :exec
DELETE FROM freight_artifacts WHERE freight_id = $1;

-- name: ListFreightArtifacts :many
SELECT a.freight_id, a.ordinal, a.artifact_type, a.subscription_name, a.version, a.metadata FROM freight_artifacts a
JOIN freight f ON f.id = a.freight_id
WHERE f.deleted_at IS NULL
ORDER BY a.freight_id, a.ordinal;
