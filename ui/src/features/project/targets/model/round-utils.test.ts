import { describe, expect, test } from 'vitest';

import { PromotionRequest } from '@ui/gen/api/v2/models';

import { collectionUpToDate, freightByOrigin, freightNames, roundPhase } from './round-utils';

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

describe('freightByOrigin()', () => {
  const ref = (warehouse: string, name: string) => ({
    name,
    origin: { kind: 'Warehouse', name: warehouse }
  });

  test('pairs each origin of the Target with the Stage, sorted by origin', () => {
    expect(
      freightByOrigin(
        {
          items: {
            'Warehouse/traefik': ref('traefik', 't1'),
            'Warehouse/coredns': ref('coredns', 'c1')
          }
        },
        {
          items: {
            'Warehouse/traefik': ref('traefik', 't2'),
            'Warehouse/coredns': ref('coredns', 'c1')
          }
        }
      ).map(({ origin, current, stage, behind }) => [origin, current?.name, stage?.name, behind])
    ).toEqual([
      ['coredns', 'c1', 'c1', false],
      ['traefik', 't1', 't2', true]
    ]);
  });

  test('an origin the Stage has added since is behind on the Target', () => {
    const [origin] = freightByOrigin(
      { items: {} },
      { items: { 'Warehouse/kyverno': ref('kyverno', 'k1') } }
    );
    expect(origin).toMatchObject({ origin: 'kyverno', current: undefined, behind: true });
  });

  test('names an origin by its key when the reference does not carry one', () => {
    const [origin] = freightByOrigin({ items: { 'Warehouse/w': { name: 'f' } } }, undefined);
    expect(origin).toMatchObject({ origin: 'w', behind: false });
  });

  test('a Target with no Freight is behind on every origin the Stage has', () => {
    expect(
      freightByOrigin(undefined, { items: { 'Warehouse/w': ref('w', 'f') } }).map(
        ({ origin, current, stage, behind }) => [origin, current, stage?.name, behind]
      )
    ).toEqual([['w', undefined, 'f', true]]);
  });

  test('nothing on either side, no origins', () => {
    expect(freightByOrigin(undefined, undefined)).toEqual([]);
  });
});
