import { describe, expect, test } from 'vitest';

import { healthSummary, healthTooltip } from './health-summary';

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

describe('healthTooltip', () => {
  test('none', () => {
    expect(healthTooltip(healthSummary(undefined), 'Stage')).toBe('No Stages');
  });

  test('one', () => {
    expect(healthTooltip(healthSummary({ count: 1 }), 'Warehouse')).toBe(
      '0 of 1 Warehouse healthy'
    );
  });

  test('several', () => {
    expect(healthTooltip(healthSummary({ count: 4, health: { healthy: 1 } }), 'Stage')).toBe(
      '1 of 4 Stages healthy'
    );
  });
});
