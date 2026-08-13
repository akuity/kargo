import { describe, expect, test } from 'vitest';

import { formatStepDuration, stepDurationMs } from './step-duration';

const START = '2026-08-13T00:00:00Z';
const startMs = new Date(START).getTime();

describe('stepDurationMs()', () => {
  test('returns null when the step has no metadata', () => {
    expect(stepDurationMs(undefined)).toBeNull();
  });
  test('returns null when the step has not started', () => {
    expect(stepDurationMs({ status: 'Running' })).toBeNull();
  });
  test('returns null for an unparseable startedAt', () => {
    expect(stepDurationMs({ startedAt: 'not a date' })).toBeNull();
  });
  test('measures a finished step from startedAt to finishedAt', () => {
    expect(stepDurationMs({ startedAt: START, finishedAt: '2026-08-13T00:00:05Z' })).toBe(5000);
  });
  test('ignores `now` for a finished step', () => {
    expect(
      stepDurationMs({ startedAt: START, finishedAt: '2026-08-13T00:00:05Z' }, startMs + 900000)
    ).toBe(5000);
  });
  test('measures a running step from startedAt to now', () => {
    expect(stepDurationMs({ startedAt: START }, startMs + 12000)).toBe(12000);
  });
  test('treats an unparseable finishedAt as still running', () => {
    expect(stepDurationMs({ startedAt: START, finishedAt: '' }, startMs + 3000)).toBe(3000);
  });
  // Clock skew between control plane and browser must not render as "-3s".
  test('returns null when now precedes startedAt', () => {
    expect(stepDurationMs({ startedAt: START }, startMs - 3000)).toBeNull();
  });
});

describe('formatStepDuration()', () => {
  // API timestamps carry second precision, so a sub-second step computes to
  // exactly 0; "0ms" would imply precision the data does not have.
  test('a zero duration renders as less than a second', () => {
    expect(formatStepDuration(0)).toBe('<1s');
  });
  test('other sub-second durations render in milliseconds', () => {
    expect(formatStepDuration(1)).toBe('1ms');
    expect(formatStepDuration(999)).toBe('999ms');
  });
  test('sub-minute durations render in seconds', () => {
    expect(formatStepDuration(1000)).toBe('1s');
    expect(formatStepDuration(1999)).toBe('1s');
    expect(formatStepDuration(59000)).toBe('59s');
  });
  test('sub-hour durations render in minutes and seconds', () => {
    expect(formatStepDuration(60000)).toBe('1m');
    expect(formatStepDuration(65000)).toBe('1m 5s');
    expect(formatStepDuration(3599000)).toBe('59m 59s');
  });
  test('longer durations render in hours and minutes', () => {
    expect(formatStepDuration(3600000)).toBe('1h');
    expect(formatStepDuration(3660000)).toBe('1h 1m');
    expect(formatStepDuration(48 * 3600000)).toBe('48h');
  });
});
