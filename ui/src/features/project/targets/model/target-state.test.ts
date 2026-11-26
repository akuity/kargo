import { describe, expect, test } from 'vitest';

import { TargetState } from '../../../../gen/api/v2/models';

import {
  describeStates,
  emptyStateCounts,
  stateOrder,
  statePrecedence,
  targetStatePresentations,
  toStateCounts,
  toTargetState,
  totalStateCount,
  worstState
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
    const counts = { ...emptyStateCounts(), Failed: 2, UpToDate: 10, Behind: 3 };
    expect(totalStateCount(counts)).toBe(15);
    expect(describeStates(counts)).toBe('10 up to date, 3 behind, 2 failed');
  });

  test('every state the server serves is presented', () => {
    expect(Object.keys(targetStatePresentations).sort()).toEqual(Object.values(TargetState).sort());
  });

  test('ranks states worst first, as the server counts a Target across Stages', () => {
    expect(worstState(['UpToDate', 'Failed', 'Behind'])).toBe('Failed');
    expect(worstState(['NotAttempted', 'Failed'])).toBe('Failed');
    expect(worstState(['Pending', 'Promoting'])).toBe('Promoting');
    expect(worstState(['UpToDate', 'Behind'])).toBe('Behind');
    expect(worstState(['NeverPromoted', 'Behind'])).toBe('NeverPromoted');
  });

  test('no governing Stage means never promoted', () => {
    expect(worstState([])).toBe('NeverPromoted');
  });

  test('reads an unknown state as never promoted, which claims the least', () => {
    expect(toTargetState('Sideways')).toBe('NeverPromoted');
    expect(toTargetState(undefined)).toBe('NeverPromoted');
    expect(toTargetState('Failed')).toBe('Failed');
  });

  test('reads a tally off the wire with every state present', () => {
    expect(toStateCounts({ Failed: 2, Sideways: 4 })).toEqual({ ...emptyStateCounts(), Failed: 2 });
    expect(toStateCounts(undefined)).toEqual(emptyStateCounts());
  });

  test('reuses the existing marks: not attempted is grey, never promoted is hollow', () => {
    expect(targetStatePresentations.NotAttempted).toMatchObject({
      label: 'Not attempted',
      colorToken: 'colorTextQuaternary'
    });
    expect(targetStatePresentations.NeverPromoted).toMatchObject({ hollow: true });
    expect(targetStatePresentations.Failed).toMatchObject({
      colorToken: 'colorError',
      attention: true
    });
  });

  test('no state speaks of health', () => {
    for (const { label } of Object.values(targetStatePresentations)) {
      expect(label.toLowerCase()).not.toMatch(/health/);
    }
  });
});
