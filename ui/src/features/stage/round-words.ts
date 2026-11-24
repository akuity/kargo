// RoundKind says which of the two a round is: a PromotionRequest or a
// PromotionRollout. The views of a round are the same for both and differ
// only in what they call it.
export type RoundKind = 'request' | 'rollout';

// roundWords is what a round is called, by kind: as a heading, as a list's
// heading, and in the middle of a sentence.
export const roundWords: Record<RoundKind, { one: string; many: string; short: string }> = {
  request: { one: 'Promotion Request', many: 'Promotion Requests', short: 'request' },
  rollout: { one: 'Promotion Rollout', many: 'Promotion Rollouts', short: 'rollout' }
};
