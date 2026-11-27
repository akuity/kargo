import { describe, expect, test } from 'vitest';

import { getOverflow, getSlideDuration } from './overflow-text-utils';

describe('getOverflow', () => {
  test('text that fits does not overflow', () => {
    expect(getOverflow(80, 100)).toBe(0);
    expect(getOverflow(100, 100)).toBe(0);
  });

  test('a sub-pixel difference is rounding, not overflow', () => {
    expect(getOverflow(100.4, 100)).toBe(0);
  });

  test('overflow is rounded up to whole pixels', () => {
    expect(getOverflow(110.2, 100)).toBe(11);
  });
});

describe('getSlideDuration', () => {
  test('a short overflow still takes the minimum time', () => {
    expect(getSlideDuration(5)).toBe(300);
  });

  test('a long overflow takes longer, at the same pace per pixel', () => {
    expect(getSlideDuration(30)).toBe(600);
    expect(getSlideDuration(60)).toBe(1200);
  });
});
