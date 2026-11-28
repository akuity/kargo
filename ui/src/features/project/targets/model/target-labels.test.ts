import { describe, expect, test } from 'vitest';

import { isInternalLabel, labelKeys, visibleLabels } from './target-labels';
import { target } from './test-fixtures';

describe('isInternalLabel()', () => {
  test.each([
    ['kargo.akuity.io/shard', true],
    ['targetset.kargo.akuity.io/owner', true],
    ['anything.kargo.akuity.io/key', true],
    ['kargo.akuity.io', true],
    ['env', false],
    ['app.kubernetes.io/name', false],
    // A domain that merely ends in the same letters is somebody else's.
    ['notkargo.akuity.io/key', false],
    ['akuity.io/key', false],
    // The domain is the part before the slash, never the value.
    ['env/kargo.akuity.io', false]
  ])('%j -> %s', (key, internal) => {
    expect(isInternalLabel(key)).toBe(internal);
  });
});

describe('visibleLabels()', () => {
  test("keeps everything but Kargo's own", () => {
    expect(
      visibleLabels(
        target('a', {
          env: 'prod',
          'kargo.akuity.io/shard': 'x',
          'targetset.kargo.akuity.io/owner': 'eks-fleet'
        })
      )
    ).toEqual([['env', 'prod']]);
  });
});

describe('labelKeys()', () => {
  test('collects distinct keys, sorted, minus Kargo-internal ones', () => {
    expect(
      labelKeys([
        target('a', {
          region: 'us',
          'kargo.akuity.io/shard': 'x',
          'targetset.kargo.akuity.io/owner': 'eks-fleet'
        }),
        target('b', { env: 'prod', region: 'eu' }),
        target('c')
      ])
    ).toEqual(['env', 'region']);
  });
});
