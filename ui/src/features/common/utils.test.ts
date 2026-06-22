import { describe, expect, test } from 'vitest';

import { Freight, FreightReference, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { getCurrentFreightForComparison } from './utils';

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
