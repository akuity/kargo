import { describe, expect, test } from 'vitest';

import { healthSummary } from './health-summary';

describe('healthSummary', () => {
  test('no stats block', () => {
    expect(healthSummary(undefined)).toEqual({ healthy: 0, count: 0, allHealthy: false });
  });

  test('empty stats block', () => {
    expect(healthSummary({})).toEqual({ healthy: 0, count: 0, allHealthy: false });
  });

  test('count without health', () => {
    expect(healthSummary({ count: 3 })).toEqual({ healthy: 0, count: 3, allHealthy: false });
  });

  test('partially healthy', () => {
    expect(healthSummary({ count: 3, health: { healthy: 2 } })).toEqual({
      healthy: 2,
      count: 3,
      allHealthy: false
    });
  });

  test('all healthy', () => {
    expect(healthSummary({ count: 3, health: { healthy: 3 } })).toEqual({
      healthy: 3,
      count: 3,
      allHealthy: true
    });
  });
});
