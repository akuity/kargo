import { describe, expect, test } from 'vitest';

import { targetRows } from './target-rows';
import { matchesSearch, parseSelector } from './target-search';
import { fleetStage, target } from './test-fixtures';

describe('parseSelector()', () => {
  test('reads comma-separated equality terms', () => {
    expect(parseSelector('tier=primary,cloud!=azure')).toEqual([
      { key: 'tier', negated: false, value: 'primary' },
      { key: 'cloud', negated: true, value: 'azure' }
    ]);
  });

  test('accepts == and surrounding space', () => {
    expect(parseSelector(' region == us ')).toEqual([
      { key: 'region', negated: false, value: 'us' }
    ]);
  });

  test.each([
    '',
    'us-east-1',
    'prod us',
    '=orphan',
    // Half-typed: reading these as "equals the empty string" would empty the
    // fleet on the way to a real selector.
    'tier=',
    'tier= ',
    'tier!=',
    'geo=us,tier='
  ])('%j is not a selector', (text) => {
    expect(parseSelector(text)).toBeUndefined();
  });
});

describe('matchesSearch()', () => {
  const us = fleetStage('prod-us', [{ matchLabels: { region: 'us' } }]);
  const [row] = targetRows(
    [target('us-east-1', { region: 'us', tier: 'canary' }, { 'prod-us': {} })],
    [us]
  );

  test.each([
    ['', true],
    ['east', true],
    ['EAST', true],
    ['canary', true],
    ['prod-us', true],
    ['eu', false]
  ])('free text %j -> %s', (needle, expected) => {
    expect(matchesSearch(row, needle)).toBe(expected);
  });

  test('free text never matches on a label the page hides', () => {
    const [hidden] = targetRows(
      [target('us-east-1', { 'targetset.kargo.akuity.io/owner': 'eks-fleet' })],
      []
    );
    expect(matchesSearch(hidden, 'eks-fleet')).toBe(false);
  });

  test('a selector naming a hidden key in full still reaches it', () => {
    const [hidden] = targetRows(
      [target('us-east-1', { 'targetset.kargo.akuity.io/owner': 'eks-fleet' })],
      []
    );
    expect(matchesSearch(hidden, 'targetset.kargo.akuity.io/owner=eks-fleet')).toBe(true);
  });

  test.each([
    ['region=us', true],
    ['region=eu', false],
    ['region=us,tier=canary', true],
    ['region=us,tier=stable', false],
    ['tier!=stable', true],
    ['tier!=canary', false],
    ['missing!=anything', true],
    ['REGION=us', false]
  ])('selector %j -> %s', (needle, expected) => {
    expect(matchesSearch(row, needle)).toBe(expected);
  });
});
