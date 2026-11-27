-- +goose Up

-- A user may abort a rollout that has not finished. The request is recorded
-- on the rollout, by whom and when, and is what the API serves as the
-- rollout's abort annotation; see AbortPromotionRollout.
--
-- What the request does depends on how far the rollout has got. A Pending
-- rollout has created no Promotions, so there is nothing to stop: the same
-- statement that records the request ends it as Aborted. That is the one
-- status write made by anything but the reconciler, and it races the
-- reconciler's claim safely, since the claim is conditional on the row as
-- read and the request advances updated_at. A Running rollout has children
-- that may still be promoting, so the request is all that is written, and
-- the reconciler, which owns a started rollout's status, stops it: it
-- creates no more children, aborts those still running, and ends the
-- rollout as Aborted once they have stopped.
--
-- Once recorded, a request is kept, and a second one changes nothing.
ALTER TABLE promotion_rollouts
    ADD COLUMN abort_requested_at TIMESTAMPTZ,
    ADD COLUMN abort_requested_by TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE promotion_rollouts
    DROP COLUMN abort_requested_by,
    DROP COLUMN abort_requested_at;
