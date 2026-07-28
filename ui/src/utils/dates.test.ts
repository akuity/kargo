import { describe, expect, test } from 'vitest';

import { parseDate } from './dates';

describe('parseDate', () => {
  test('returns undefined for missing values', () => {
    expect(parseDate(undefined)).toBeUndefined();
    expect(parseDate('')).toBeUndefined();
  });

  test('returns undefined for unparseable values', () => {
    expect(parseDate('not a date')).toBeUndefined();
  });

  test('parses a valid ISO 8601 timestamp', () => {
    const date = parseDate('2026-06-23T12:34:56Z');
    expect(date).toBeInstanceOf(Date);
    expect(date?.toISOString()).toBe('2026-06-23T12:34:56.000Z');
  });

  // Regression guard: a bare `new Date(x || '')` yields a *truthy* Invalid Date,
  // which defeats `date ? format(date) : ''` guards and makes date-fns throw.
  test('never returns a truthy Invalid Date', () => {
    const date = parseDate('');
    expect(date ? 'truthy' : 'falsy').toBe('falsy');
  });
});
