import { describe, expect, test } from 'vitest';

import { buildLabelSelector } from './label-selector';

describe('buildLabelSelector', () => {
  test('one value per key', () => {
    expect(buildLabelSelector(['cloud=aws', 'team=core'])).toBe('cloud=aws,team=core');
  });

  test("values of one key are OR'd, keys AND'd", () => {
    expect(buildLabelSelector(['cloud=aws', 'cloud=gcp', 'team=core'])).toBe(
      'cloud in (aws,gcp),team=core'
    );
  });

  test('nothing selected', () => {
    expect(buildLabelSelector([])).toBe('');
  });
});
