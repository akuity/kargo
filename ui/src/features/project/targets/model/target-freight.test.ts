import { describe, expect, test } from 'vitest';

import { entryCollection, freightByOrigin, freightNames } from './target-freight';

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

describe('entryCollection()', () => {
  const origin = { kind: 'Warehouse', name: 'web' };

  test('keys each piece of Freight by its origin, as a Stage keys its own', () => {
    expect(entryCollection([{ name: 'f1', origin }])).toEqual({
      items: { 'Warehouse/web': { name: 'f1', origin } }
    });
  });

  test('fills in what the Freight carries when it can be resolved', () => {
    const images = [{ repoURL: 'ghcr.io/example/web', tag: '1.0.0' }];
    const collection = entryCollection([{ name: 'f1', origin }], (name) =>
      name === 'f1' ? { name, images } : undefined
    );
    expect(collection?.items?.['Warehouse/web']).toEqual({ name: 'f1', origin, images });
  });

  test('pairs with the Stage origin by origin', () => {
    const stage = { items: { 'Warehouse/web': { name: 'f2', origin } } };
    expect(freightByOrigin(entryCollection([{ name: 'f1', origin }]), stage)).toMatchObject([
      { origin: 'web', behind: true }
    ]);
  });

  test('a Target with no Freight has no collection', () => {
    expect(entryCollection([])).toBeUndefined();
    expect(entryCollection(undefined)).toBeUndefined();
  });
});
