-- +goose Up

-- Serve reading where a Target stands with a Stage that governs it; see
-- ListTargetStatuses. No table records that. It is read from these rows on
-- request, one probe per Target and Stage, and these two indexes are what
-- keep a probe from reading further into the history than its answer.

-- A Stage's Promotions to one Target, newest first: the last one, and the
-- last that succeeded. The index of a Target's Promotions cannot serve this,
-- because it holds every Stage's: a Stage that has never promoted to a
-- Target would read the Target's whole history to learn so, and one that
-- promotes rarely would read past every Promotion another Stage has made
-- since.
CREATE INDEX promotions_stage_target_created_at_idx
    ON promotions (stage_id, target_name, created_at DESC, id DESC)
    WHERE target_name IS NOT NULL;

-- A Stage's rollouts that finished without succeeding, by number: the only
-- ones that can have ended before attempting a Target they name. Few of a
-- Stage's rollouts end this way, so a Target with no Promotion yet does not
-- read every rollout the Stage has had to learn that none named it. The
-- predicate must match the query's word for word.
CREATE INDEX promotion_rollouts_unsuccessful_idx
    ON promotion_rollouts (stage_id, number)
    WHERE status_phase IN ('Failed', 'Errored', 'Aborted');

-- +goose Down
DROP INDEX promotion_rollouts_unsuccessful_idx;
DROP INDEX promotions_stage_target_created_at_idx;
