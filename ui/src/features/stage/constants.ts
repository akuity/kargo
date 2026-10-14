// PromotionRollouts live in the database, which offers no watch, so lists of
// them are polled at this interval.
export const PROMOTION_ROLLOUTS_POLL_INTERVAL_MS = 5_000;
