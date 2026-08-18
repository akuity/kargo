import { describe, expect, test } from 'vitest';

import { elapsedStepDurationMs, formatStepDuration } from './step-duration';

const START = '2026-08-13T00:00:00Z';
const startMs = new Date(START).getTime();

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

describe('elapsedStepDurationMs()', () => {
  test('returns null when the step has no metadata', () => {
    expect(elapsedStepDurationMs(undefined)).toBeNull();
  });
  test('returns null when the step has not started', () => {
    expect(elapsedStepDurationMs({ status: 'Running' })).toBeNull();
  });
  test('returns null for an unparseable startedAt', () => {
    expect(elapsedStepDurationMs({ startedAt: 'not a date' })).toBeNull();
  });
  test('measures from startedAt to now', () => {
    expect(elapsedStepDurationMs({ startedAt: START }, startMs + 12 * SECOND)).toBe(12 * SECOND);
  });
  // The caller only mounts the ticking component for a step with no finishedAt,
  // but the function ignores it either way rather than quietly disagreeing.
  test('ignores finishedAt', () => {
    expect(
      elapsedStepDurationMs(
        { startedAt: START, finishedAt: '2026-08-13T00:00:05Z' },
        startMs + 3 * SECOND
      )
    ).toBe(3 * SECOND);
  });
  // Clock skew between control plane and browser must not render as "-3s".
  test('returns null when now precedes startedAt', () => {
    expect(elapsedStepDurationMs({ startedAt: START }, startMs - 3 * SECOND)).toBeNull();
  });
});

describe('formatStepDuration()', () => {
  // API timestamps carry second precision, so a sub-second step computes to
  // exactly 0; "0ms" would imply precision the data does not have.
  test('a zero duration renders as less than a second', () => {
    expect(formatStepDuration(0)).toBe('<1s');
  });
  // Sub-second values bypass date-fns entirely -- intervalToDuration truncates
  // them to an empty Duration, which formatDuration renders as ''.
  test('other sub-second durations render in milliseconds', () => {
    expect(formatStepDuration(1)).toBe('1ms');
    expect(formatStepDuration(999)).toBe('999ms');
  });
  test('sub-minute durations render in seconds', () => {
    expect(formatStepDuration(SECOND)).toBe('1s');
    expect(formatStepDuration(1999)).toBe('1s');
    expect(formatStepDuration(59 * SECOND)).toBe('59s');
  });
  test('sub-hour durations render in minutes and seconds', () => {
    expect(formatStepDuration(MINUTE)).toBe('1m');
    expect(formatStepDuration(MINUTE + 5 * SECOND)).toBe('1m 5s');
    expect(formatStepDuration(HOUR - SECOND)).toBe('59m 59s');
  });
  test('hour-scale durations render in hours and minutes', () => {
    expect(formatStepDuration(HOUR)).toBe('1h');
    expect(formatStepDuration(HOUR + MINUTE)).toBe('1h 1m');
    expect(formatStepDuration(HOUR + 40 * MINUTE)).toBe('1h 40m');
  });
  // Above an hour, seconds are noise rather than precision.
  test('drops seconds from durations of an hour or more', () => {
    expect(formatStepDuration(HOUR + 30 * SECOND)).toBe('1h');
    expect(formatStepDuration(DAY + HOUR + MINUTE + SECOND)).toBe('1d 1h 1m');
  });
  // date-fns normalizes whole days out of the hours field, so a two-day step
  // reads "2d" rather than "48h".
  test('day-scale durations render in days', () => {
    expect(formatStepDuration(DAY)).toBe('1d');
    expect(formatStepDuration(2 * DAY)).toBe('2d');
    expect(formatStepDuration(DAY + HOUR)).toBe('1d 1h');
  });
  // intervalToDuration rolls 40 days up into {months: 1, days: 9}. The coarse
  // units must all be formatted or the largest component vanishes.
  test('does not drop the largest unit of a very long duration', () => {
    expect(formatStepDuration(40 * DAY)).toBe('1mo 9d');
  });
  // Zero-valued units are omitted rather than padded, so nothing renders "0m".
  test('omits zero-valued units', () => {
    expect(formatStepDuration(2 * MINUTE)).toBe('2m');
    expect(formatStepDuration(2 * HOUR)).toBe('2h');
  });
});
