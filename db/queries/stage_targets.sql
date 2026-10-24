-- The Project's target-aware Stages with their selectors. A classic Stage,
-- whose selectors are NULL, governs nothing and is left out.
-- name: ListStageSelectors :many
SELECT id, name, target_selectors
FROM stages
WHERE project_name = sqlc.arg(project_name) AND target_selectors IS NOT NULL
ORDER BY id;

-- The Project's Targets with what a selector reads: their labels.
-- name: ListTargetLabels :many
SELECT id, name, labels
FROM targets
WHERE project_name = sqlc.arg(project_name)
ORDER BY id;

-- The Project's governance as it stands.
-- name: ListStageTargets :many
SELECT stage_targets.stage_id, stage_targets.target_id
FROM stage_targets
JOIN stages ON stages.id = stage_targets.stage_id
WHERE stages.project_name = sqlc.arg(project_name)
ORDER BY stage_targets.stage_id, stage_targets.target_id;

-- Records pairs, given as two arrays of equal length. A pair whose Stage or
-- Target has gone since it was read is skipped rather than failing on its
-- foreign key: the next reconcile of the Project settles it.
-- name: InsertStageTargets :exec
INSERT INTO stage_targets (stage_id, target_id)
SELECT pair.stage_id, pair.target_id
FROM (
    SELECT
        unnest(sqlc.arg(stage_ids)::text[]) AS stage_id,
        unnest(sqlc.arg(target_ids)::uuid[]) AS target_id
) AS pair
JOIN stages ON stages.id = pair.stage_id
JOIN targets ON targets.id = pair.target_id
ON CONFLICT DO NOTHING;

-- Removes pairs, given as two arrays of equal length.
-- name: DeleteStageTargets :exec
DELETE FROM stage_targets
USING (
    SELECT
        unnest(sqlc.arg(stage_ids)::text[]) AS stage_id,
        unnest(sqlc.arg(target_ids)::uuid[]) AS target_id
) AS pair
WHERE stage_targets.stage_id = pair.stage_id AND stage_targets.target_id = pair.target_id;

-- What a Project's governance depends on, condensed per Project so that a
-- poll can tell which Projects changed without reading their rows: a digest
-- of its Stages' identities and selectors, and its Targets' count and latest
-- write. Adding, removing, relabeling or reselecting anything changes one of
-- them. A Project with Stages but no Targets, or the reverse, appears with
-- the other side empty.
-- name: ListStageTargetInputs :many
SELECT
    COALESCE(stage_digests.project_name, target_digests.project_name)::text AS project_name,
    COALESCE(stage_digests.digest, '')::text AS stage_digest,
    COALESCE(target_digests.count, 0)::bigint AS target_count,
    COALESCE(target_digests.updated_at, 'epoch'::timestamptz)::timestamptz AS targets_updated_at
FROM (
    SELECT
        project_name,
        md5(string_agg(id || '=' || COALESCE(target_selectors::text, ''), ',' ORDER BY id)) AS digest
    FROM stages
    GROUP BY project_name
) AS stage_digests
FULL JOIN (
    SELECT project_name, count(*) AS count, max(updated_at) AS updated_at
    FROM targets
    GROUP BY project_name
) AS target_digests ON target_digests.project_name = stage_digests.project_name
ORDER BY 1;
