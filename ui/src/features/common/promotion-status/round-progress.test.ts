import { describe, expect, test } from 'vitest';

import { describeRound, roundBadgePhase, roundBadgeText, roundSegments } from './round-progress';

describe('roundSegments()', () => {
  test('an absent or empty summary has no segments', () => {
    expect(roundSegments(undefined)).toEqual({ total: 0, segments: [] });
    expect(roundSegments({})).toEqual({ total: 0, segments: [] });
    expect(roundSegments({ succeeded: 0, failed: 0 })).toEqual({ total: 0, segments: [] });
  });

  test('a round that entirely succeeded is one full-width segment', () => {
    expect(roundSegments({ succeeded: 4 })).toEqual({
      total: 4,
      segments: [{ phase: 'Succeeded', count: 4, percent: 100 }]
    });
  });

  test('a round that has not started is one pending segment', () => {
    expect(roundSegments({ pending: 7 })).toEqual({
      total: 7,
      segments: [{ phase: 'Pending', count: 7, percent: 100 }]
    });
  });

  test('segments follow presentation order and omit empty phases', () => {
    const { total, segments } = roundSegments({
      errored: 1,
      pending: 2,
      succeeded: 5,
      running: 2,
      aborted: 0
    });
    expect(total).toBe(10);
    expect(segments.map((s) => s.phase)).toEqual(['Succeeded', 'Running', 'Pending', 'Errored']);
    expect(segments.map((s) => s.percent)).toEqual([50, 20, 20, 10]);
  });

  test('percents sum to 100 when counts do not divide evenly', () => {
    const { segments } = roundSegments({ succeeded: 1, failed: 1, errored: 1 });
    const sum = segments.reduce((acc, s) => acc + s.percent, 0);
    expect(sum).toBeCloseTo(100, 6);
    expect(segments.every((s) => s.percent > 33 && s.percent < 34)).toBe(true);
  });

  test('negative counts are treated as zero', () => {
    expect(roundSegments({ succeeded: 2, failed: -3 })).toEqual({
      total: 2,
      segments: [{ phase: 'Succeeded', count: 2, percent: 100 }]
    });
  });
});

describe('describeRound()', () => {
  test('empty round', () => {
    expect(describeRound(roundSegments({}))).toBe('No Targets in this round');
  });

  test('lists each phase in order with the total', () => {
    expect(describeRound(roundSegments({ succeeded: 3, errored: 1 }))).toBe(
      '3 succeeded, 1 errored of 4 Targets'
    );
  });

  test('singular Target', () => {
    expect(describeRound(roundSegments({ running: 1 }))).toBe('1 running of 1 Target');
  });
});

describe('roundBadgePhase()', () => {
  test('an empty round has no phase', () => {
    expect(roundBadgePhase(roundSegments({}))).toBeUndefined();
  });

  test.each([
    [{ succeeded: 3 }, 'Succeeded'],
    [{ succeeded: 3, pending: 1 }, 'Pending'],
    [{ succeeded: 3, running: 1, pending: 1 }, 'Running'],
    [{ succeeded: 3, running: 1, failed: 1 }, 'Failed'],
    [{ succeeded: 3, failed: 1, errored: 1 }, 'Errored'],
    [{ succeeded: 3, aborted: 1 }, 'Aborted'],
    [{ aborted: 2, pending: 1 }, 'Pending']
  ])('%j -> %s', (summary, phase) => {
    expect(roundBadgePhase(roundSegments(summary))).toBe(phase);
  });
});

describe('roundBadgeText()', () => {
  test('succeeded over total', () => {
    expect(roundBadgeText(roundSegments({ succeeded: 37, errored: 3 }))).toBe('37/40');
    expect(roundBadgeText(roundSegments({ pending: 5 }))).toBe('0/5');
    expect(roundBadgeText(roundSegments({}))).toBe('0/0');
  });
});
