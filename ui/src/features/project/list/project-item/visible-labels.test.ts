import { describe, expect, test } from 'vitest';

import { fitCount, visibleLabels } from './visible-labels';

describe('visibleLabels', () => {
  test('drops system labels and sorts by key', () => {
    expect(
      visibleLabels({
        team: 'core',
        'app.kubernetes.io/managed-by': 'Helm',
        'kargo.akuity.io/project': 'true',
        'argocd.argoproj.io/instance': 'x',
        'example.com/tier': 'gold',
        cloud: 'aws'
      })
    ).toEqual([
      ['cloud', 'aws'],
      ['example.com/tier', 'gold'],
      ['team', 'core']
    ]);
  });

  test('handles a Project without labels', () => {
    expect(visibleLabels(undefined)).toEqual([]);
  });
});

describe('fitCount', () => {
  test('everything fits', () => {
    expect(fitCount([50, 50], 30, 104, 4)).toBe(2);
  });

  test('keeps room for the +N tag', () => {
    // 50 + 4 + 50 = 104 would fit, but then +N (30) must follow: 104 + 4 + 30 > 120.
    expect(fitCount([50, 50, 50], 30, 120, 4)).toBe(1);
  });

  test('a first tag that does not fit is still shown (CSS truncates it)', () => {
    expect(fitCount([200, 50], 30, 100, 4)).toBe(1);
  });

  test('no labels', () => {
    expect(fitCount([], 30, 100, 4)).toBe(0);
  });
});
