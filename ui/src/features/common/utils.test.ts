import { create } from '@bufbuild/protobuf';
import { describe, expect, it, test } from 'vitest';

import {
  Freight,
  FreightReference,
  FreightSchema,
  Stage,
  StageSchema
} from '@ui/gen/api/v1alpha1/generated_pb';

import {
  ALIAS_LABEL_KEY,
  getCurrentFreightByWarehouse,
  getCurrentFreightForComparison,
  reconstructFreightFromHistory,
  getShortFreightLabel
} from './utils';

const ref = (origin: string, image: string): FreightReference =>
  ({
    origin: { kind: 'Warehouse', name: origin },
    images: [{ repoURL: image }]
  }) as unknown as FreightReference;

// stage builds a Stage whose current freight collection holds one piece of
// Freight per Warehouse, keyed by origin name (mirroring the API shape).
const stage = (refs: FreightReference[]): Stage =>
  ({
    status: {
      freightHistory: [
        {
          items: Object.fromEntries(refs.map((r) => [r.origin?.name ?? '', r]))
        }
      ]
    }
  }) as unknown as Stage;

const incoming = (origin: string): Freight =>
  ({ origin: { kind: 'Warehouse', name: origin } }) as unknown as Freight;

describe('getCurrentFreightForComparison', () => {
  test('multi-warehouse: matches the current Freight from the incoming origin', () => {
    const current = stage([
      ref('warehouse-a', 'ghcr.io/acme/a'),
      ref('warehouse-b', 'ghcr.io/acme/b')
    ]);

    const result = getCurrentFreightForComparison(current, incoming('warehouse-b'));
    expect(result?.origin?.name).toBe('warehouse-b');
  });

  test('does not default to the first warehouse when the incoming origin differs', () => {
    const current = stage([
      ref('warehouse-a', 'ghcr.io/acme/a'),
      ref('warehouse-b', 'ghcr.io/acme/b')
    ]);

    // Incoming Freight originates from a Warehouse the Stage has no Freight
    // from yet -- there is nothing to compare against, so the result is
    // undefined rather than warehouse-a (the regression this fix addresses).
    const result = getCurrentFreightForComparison(current, incoming('warehouse-c'));
    expect(result).toBeUndefined();
  });

  test('single-warehouse: matches the only current Freight', () => {
    const current = stage([ref('warehouse-a', 'ghcr.io/acme/a')]);

    const result = getCurrentFreightForComparison(current, incoming('warehouse-a'));
    expect(result?.origin?.name).toBe('warehouse-a');
  });

  test('stage with no freight history → undefined', () => {
    const empty = { status: { freightHistory: [] } } as unknown as Stage;

    expect(getCurrentFreightForComparison(empty, incoming('warehouse-a'))).toBeUndefined();
  });
});

describe('getCurrentFreightByWarehouse', () => {
  it('returns an empty map when the stage has no freight history', () => {
    const stage = create(StageSchema, { metadata: { name: 'test' } });
    expect(getCurrentFreightByWarehouse(stage)).toEqual({});
  });

  it('keys the current freight by warehouse identifier', () => {
    const stage = create(StageSchema, {
      status: {
        freightHistory: [
          {
            items: {
              'Warehouse/w-1': { name: 'freight-aaa' },
              'Warehouse/w-2': { name: 'freight-bbb' }
            }
          },
          // older collection -- must be ignored
          { items: { 'Warehouse/w-1': { name: 'freight-old' } } }
        ]
      }
    });

    const result = getCurrentFreightByWarehouse(stage);

    expect(Object.keys(result).sort()).toEqual(['Warehouse/w-1', 'Warehouse/w-2']);
    expect(result['Warehouse/w-1'].reference.name).toBe('freight-aaa');
    expect(result['Warehouse/w-2'].reference.name).toBe('freight-bbb');
  });

  it('resolves the alias from the freight map, preferring the alias field', () => {
    const stage = create(StageSchema, {
      status: {
        freightHistory: [{ items: { 'Warehouse/w-1': { name: 'freight-aaa' } } }]
      }
    });
    const freightMap: Record<string, Freight> = {
      'freight-aaa': create(FreightSchema, { alias: 'tasty-tiger' })
    };

    expect(getCurrentFreightByWarehouse(stage, freightMap)['Warehouse/w-1'].alias).toBe(
      'tasty-tiger'
    );
  });

  it('falls back to the alias label when the alias field is empty', () => {
    const stage = create(StageSchema, {
      status: {
        freightHistory: [{ items: { 'Warehouse/w-1': { name: 'freight-aaa' } } }]
      }
    });
    const freightMap: Record<string, Freight> = {
      'freight-aaa': create(FreightSchema, {
        metadata: { labels: { [ALIAS_LABEL_KEY]: 'brave-bear' } }
      })
    };

    expect(getCurrentFreightByWarehouse(stage, freightMap)['Warehouse/w-1'].alias).toBe(
      'brave-bear'
    );
  });

  it('leaves the alias undefined when the freight is not in the map', () => {
    const stage = create(StageSchema, {
      status: {
        freightHistory: [{ items: { 'Warehouse/w-1': { name: 'freight-aaa' } } }]
      }
    });

    expect(getCurrentFreightByWarehouse(stage, {})['Warehouse/w-1'].alias).toBeUndefined();
  });
});

describe('getShortFreightLabel', () => {
  it('truncates the hash to seven characters when there is no alias', () => {
    expect(getShortFreightLabel('abcdef0123456789')).toBe('abcdef0');
  });

  it('combines the alias with the short hash when an alias is provided', () => {
    expect(getShortFreightLabel('abcdef0123456789', 'tasty-tiger')).toBe('tasty-tiger (abcdef0)');
  });

  it('returns an empty string when the name is missing', () => {
    expect(getShortFreightLabel()).toBe('');
  });
});

describe('reconstructFreightFromHistory', () => {
  it('preserves the historical freight contents when synthesizing a Freight', () => {
    const reference = ref('warehouse-a', 'ghcr.io/acme/a');
    const freight = reconstructFreightFromHistory(reference, 'test-project');

    expect(freight).toMatchObject({
      kind: 'Freight',
      apiVersion: 'kargo.akuity.io/v1alpha1',
      metadata: {
        name: undefined,
        namespace: 'test-project'
      },
      origin: { kind: 'Warehouse', name: 'warehouse-a' },
      images: [{ repoURL: 'ghcr.io/acme/a' }]
    });
  });
});
