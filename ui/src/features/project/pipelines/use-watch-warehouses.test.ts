import { describe, expect, it } from 'vitest';

import { Warehouse } from '@ui/gen/api/v2/models';

import { pendingRefreshes } from './use-watch-warehouses';

const warehouse = (name: string, refreshRequest?: string, lastHandledRefresh?: string) =>
  ({
    metadata: {
      name,
      annotations: refreshRequest ? { 'kargo.akuity.io/refresh': refreshRequest } : undefined
    },
    status: { lastHandledRefresh }
  }) as Warehouse;

describe('pendingRefreshes', () => {
  it('returns nothing for no Warehouses', () => {
    expect(pendingRefreshes([])).toEqual({});
  });

  it('ignores a Warehouse with no refresh request', () => {
    expect(pendingRefreshes([warehouse('a')])).toEqual({});
  });

  it('ignores a Warehouse that handled its refresh request', () => {
    expect(pendingRefreshes([warehouse('a', 'r1', 'r1')])).toEqual({});
  });

  it('marks a Warehouse with an unhandled refresh request', () => {
    expect(
      pendingRefreshes([
        warehouse('a', 'r2', 'r1'),
        warehouse('b', 'r1'),
        warehouse('c', 'r1', 'r1')
      ])
    ).toEqual({ a: true, b: true });
  });
});
