import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import {
  filterToWarehouse,
  getFocusableWarehouses,
  getFocusedWarehouse,
  getHeaderShift,
  groupByWarehouse,
  takeFreight
} from './freight-warehouse-group-utils';

const freightFrom = (warehouse: string, name: string): Freight => ({
  metadata: { name, uid: `uid-${name}` },
  origin: { kind: 'Warehouse', name: warehouse }
});

const names = (freights: Freight[]) => freights.map((f) => f.metadata?.name);

describe('groupByWarehouse', () => {
  test('returns no groups for no freight', () => {
    expect(groupByWarehouse([])).toEqual([]);
  });

  test('groups consecutive freight from the same Warehouse', () => {
    const groups = groupByWarehouse([
      freightFrom('a', '1'),
      freightFrom('a', '2'),
      freightFrom('b', '3')
    ]);

    expect(groups.map((g) => g.warehouse)).toEqual(['a', 'b']);
    expect(names(groups[0].freights)).toEqual(['1', '2']);
    expect(names(groups[1].freights)).toEqual(['3']);
  });

  test('starts a new run each time a Warehouse comes back', () => {
    const groups = groupByWarehouse([
      freightFrom('a', '1'),
      freightFrom('b', '2'),
      freightFrom('a', '3')
    ]);

    expect(groups.map((g) => g.warehouse)).toEqual(['a', 'b', 'a']);
  });

  test('keys each group by its oldest freight', () => {
    const groups = groupByWarehouse([
      freightFrom('a', '2'),
      freightFrom('a', '3'),
      freightFrom('b', '4')
    ]);

    expect(groups.map((g) => g.key)).toEqual(['a-uid-3', 'b-uid-4']);
  });

  test('keeps group keys while freight from any Warehouse arrives at the front', () => {
    const before = groupByWarehouse([
      freightFrom('a', '3'),
      freightFrom('b', '4'),
      freightFrom('a', '5')
    ]);
    const keys = before.map((g) => g.key);

    const sameWarehouse = groupByWarehouse([
      freightFrom('a', '2'),
      ...before.flatMap((g) => g.freights)
    ]);
    const otherWarehouse = groupByWarehouse([
      freightFrom('b', '2'),
      ...before.flatMap((g) => g.freights)
    ]);

    expect(sameWarehouse.map((g) => g.key)).toEqual(keys);
    expect(otherWarehouse.map((g) => g.key).slice(1)).toEqual(keys);
  });

  test('groups freight without an origin together', () => {
    const groups = groupByWarehouse([{ metadata: { name: '1' } }, { metadata: { name: '2' } }]);

    expect(groups).toHaveLength(1);
    expect(groups[0].warehouse).toBe('');
  });
});

describe('getHeaderShift', () => {
  // a group of three 100px cards with 4px gaps, 200px into the strip
  const left = 200;
  const width = 308;
  const cardWidth = 100;

  test('stays put while the group starts in view', () => {
    expect(getHeaderShift(0, left, width, cardWidth)).toBe(0);
    expect(getHeaderShift(200, left, width, cardWidth)).toBe(0);
  });

  test('follows the first card in view when paged partway into the group', () => {
    expect(getHeaderShift(304, left, width, cardWidth)).toBe(104);
  });

  test('stops at the last card', () => {
    expect(getHeaderShift(1000, left, width, cardWidth)).toBe(208);
  });
});

describe('getFocusedWarehouse', () => {
  test('returns the Warehouse the timeline is narrowed to', () => {
    expect(getFocusedWarehouse('a', false)).toBe('a');
  });

  test('shows every Warehouse while promoting', () => {
    expect(getFocusedWarehouse('a', true)).toBe('');
  });
});

describe('filterToWarehouse', () => {
  const freights = [freightFrom('a', '1'), freightFrom('b', '2'), freightFrom('a', '3')];

  test('keeps only the freight from the Warehouse', () => {
    expect(names(filterToWarehouse(freights, 'a'))).toEqual(['1', '3']);
  });

  test('keeps all freight when no Warehouse is given', () => {
    expect(filterToWarehouse(freights, '')).toBe(freights);
  });
});

describe('getFocusableWarehouses', () => {
  test('offers every Warehouse when the pipeline shows them all', () => {
    expect(getFocusableWarehouses(['a', 'b', 'c'], [])).toEqual(['a', 'b', 'c']);
  });

  test('offers only the Warehouses the pipeline shows', () => {
    expect(getFocusableWarehouses(['a', 'b', 'c'], ['c', 'a'])).toEqual(['a', 'c']);
  });
});

describe('takeFreight', () => {
  const groups = groupByWarehouse([
    freightFrom('a', '1'),
    freightFrom('a', '2'),
    freightFrom('b', '3'),
    freightFrom('b', '4')
  ]);

  test('keeps the first freight and the groups they fall in', () => {
    const taken = takeFreight(groups, 3);

    expect(taken.map((g) => g.warehouse)).toEqual(['a', 'b']);
    expect(names(taken[1].freights)).toEqual(['3']);
  });

  test('keeps group keys', () => {
    expect(takeFreight(groups, 3).map((g) => g.key)).toEqual(groups.map((g) => g.key));
  });

  test('keeps everything when the count covers it', () => {
    expect(takeFreight(groups, 10)).toEqual(groups);
  });
});
