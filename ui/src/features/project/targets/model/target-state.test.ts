import { describe, expect, test } from 'vitest';

import {
  describeStates,
  emptyStateCounts,
  stateOrder,
  statePrecedence,
  targetStatePresentations,
  totalStateCount
} from './target-state';

describe('target states', () => {
  test('every state is presented, ordered, and ranked exactly once', () => {
    const states = Object.keys(targetStatePresentations).sort();
    expect([...stateOrder].sort()).toEqual(states);
    expect([...statePrecedence].sort()).toEqual(states);
  });

  test('an empty tally has every state at zero', () => {
    const counts = emptyStateCounts();
    expect(totalStateCount(counts)).toBe(0);
    expect(describeStates(counts)).toBe('no Targets');
  });

  test('a tally reads back in the order bars draw it, skipping empty states', () => {
    const counts = { ...emptyStateCounts(), Unhealthy: 2, Healthy: 10, Behind: 3 };
    expect(totalStateCount(counts)).toBe(15);
    expect(describeStates(counts)).toBe('10 healthy, 3 behind, 2 unhealthy');
  });
});
