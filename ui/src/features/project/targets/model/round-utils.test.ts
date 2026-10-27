import { describe, expect, test } from 'vitest';

import { PromotionRequest } from '@ui/gen/api/v2/models';

import { collectionUpToDate, freightNames, roundPhase } from './round-utils';

const request = (targets: string[], status?: PromotionRequest['status']): PromotionRequest => ({
  metadata: { name: 'prod.01a.x' },
  spec: { stage: 'prod', freight: 'f1', targets: targets.map((t) => ({ name: t })) },
  status
});

describe('roundPhase()', () => {
  test('a Target the round never named has no phase', () => {
    expect(roundPhase(request(['other']), 't')).toBeUndefined();
    expect(roundPhase(undefined, 't')).toBeUndefined();
  });

  test("takes the child Promotion's phase when there is one", () => {
    const round = request(['t'], { targets: [{ name: 't', phase: 'Running' }] });
    expect(roundPhase(round, 't')).toBe('Running');
  });

  test('a child without a phase yet is Pending', () => {
    const round = request(['t'], { targets: [{ name: 't' }] });
    expect(roundPhase(round, 't')).toBe('Pending');
  });

  test.each([
    // A round that ended before creating a child ended its Targets with it.
    ['Succeeded', 'Succeeded'],
    ['Failed', 'Failed'],
    ['Errored', 'Errored'],
    // Aborted is terminal too, mirroring PromotionRequestPhase.IsTerminal:
    // reading it as Pending would leave the fleet looking mid-promotion.
    ['Aborted', 'Aborted'],
    ['Running', 'Pending'],
    [undefined, 'Pending']
  ])('a round in %s with no child reports %s', (phase, expected) => {
    expect(roundPhase(request(['t'], { phase }), 't')).toBe(expected);
  });
});

describe('collectionUpToDate()', () => {
  test.each([
    [{ id: 'abc' }, 'abc', true],
    [{ id: 'old' }, 'abc', false],
    // Nothing to compare: the question is open, not answered false.
    [undefined, 'abc', undefined],
    [{ id: 'abc' }, undefined, undefined],
    [{}, 'abc', undefined]
  ])('%j against %j -> %s', (collection, collectionID, expected) => {
    expect(collectionUpToDate(collection, collectionID)).toBe(expected);
  });
});

describe('freightNames()', () => {
  test('lists each origin Freight by name, sorted', () => {
    expect(
      freightNames({
        id: 'c',
        items: { 'Warehouse/b': { name: 'zz' }, 'Warehouse/a': { name: 'aa' } }
      })
    ).toEqual(['aa', 'zz']);
  });

  test('handles an absent collection', () => {
    expect(freightNames(undefined)).toEqual([]);
  });
});
