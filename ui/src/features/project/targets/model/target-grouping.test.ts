import { describe, expect, test } from 'vitest';

import { UNLABELED, groupHeading, groupTargetRows, groupTitle, splitKeys } from './target-grouping';
import { targetRows } from './target-rows';
import { target } from './test-fixtures';

describe('groupTargetRows()', () => {
  const rows = targetRows(
    [
      target('us-west-2', { region: 'us' }),
      target('eu-central-1', { region: 'eu' }),
      target('orphan'),
      target('us-east-1', { region: 'us' })
    ],
    []
  );

  test('without a key, one group', () => {
    expect(groupTargetRows(rows)).toHaveLength(1);
    expect(groupTargetRows(rows)[0].rows).toHaveLength(4);
  });

  test('by label, unlabeled last', () => {
    const groups = groupTargetRows(rows, 'region');
    expect(groups.map((g) => g.value)).toEqual(['eu', 'us', UNLABELED]);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['us-east-1', 'us-west-2']);
  });

  test('an empty label value groups with the unlabeled, not into a blank group', () => {
    const withEmpty = targetRows(
      [target('a', { region: '' }), target('b', { region: 'us' }), target('c')],
      []
    );
    const groups = groupTargetRows(withEmpty, 'region');
    expect(groups.map((g) => g.value)).toEqual(['us', UNLABELED]);
    expect(groups[1].rows.map((r) => r.target.metadata?.name)).toEqual(['a', 'c']);
  });

  test('no rows yields no groups', () => {
    expect(groupTargetRows([])).toEqual([]);
    expect(groupTargetRows([], 'region')).toEqual([]);
  });
});

describe('groupHeading() and groupTitle()', () => {
  const group = (value: string) => ({ value, rows: [] });

  test.each([
    [group('us'), 'geo', 'us', 'geo=us'],
    [group(UNLABELED), 'geo', 'no geo label', 'no geo label'],
    [group(''), undefined, 'All Targets', 'All Targets']
  ])('%j by %j', (target, key, heading, title) => {
    expect(groupHeading(target, key)).toBe(heading);
    expect(groupTitle(target, key)).toBe(title);
  });
});

describe('splitKeys()', () => {
  test('a short list all fits on the row', () => {
    expect(splitKeys(['geo', 'cloud', 'tier'])).toEqual({
      inline: ['geo', 'cloud', 'tier'],
      overflow: []
    });
  });

  test('the tail past four folds into the dropdown', () => {
    expect(splitKeys(['geo', 'cloud', 'tier', 'env', 'stage', 'shard'])).toEqual({
      inline: ['geo', 'cloud', 'tier', 'env'],
      overflow: ['stage', 'shard']
    });
  });

  test('a long key folds in however early it comes', () => {
    expect(splitKeys(['app.kubernetes.io/part-of', 'geo', 'cloud'])).toEqual({
      inline: ['geo', 'cloud'],
      overflow: ['app.kubernetes.io/part-of']
    });
  });

  test('short keys further down are promoted past it', () => {
    expect(
      splitKeys(['app.kubernetes.io/part-of', 'geo', 'cloud', 'tier', 'env', 'stage'])
    ).toEqual({
      inline: ['geo', 'cloud', 'tier', 'env'],
      overflow: ['app.kubernetes.io/part-of', 'stage']
    });
  });

  test('when every key is long the order stands', () => {
    const keys = ['app.kubernetes.io/part-of', 'app.kubernetes.io/component'];
    expect(splitKeys(keys)).toEqual({ inline: keys, overflow: [] });
  });

  test('no keys, nothing to show', () => {
    expect(splitKeys([])).toEqual({ inline: [], overflow: [] });
  });
});
