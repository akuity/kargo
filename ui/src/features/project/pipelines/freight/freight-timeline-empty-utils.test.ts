import { describe, expect, test } from 'vitest';

import { EmptyVariant, getVariant } from './freight-timeline-empty-utils';

describe('getVariant()', () => {
  const base = {
    promotionMode: false,
    filtered: false,
    hasWarehouses: true,
    hasWarehouseController: true
  };

  const testCases: {
    name: string;
    args: Parameters<typeof getVariant>[0];
    expected: EmptyVariant;
  }[] = [
    {
      name: 'promotion mode wins over everything',
      args: { ...base, promotionMode: true, filtered: true, hasWarehouseController: false },
      expected: 'promotion'
    },
    {
      name: 'active filters explain the absence',
      args: { ...base, filtered: true, hasWarehouseController: false },
      expected: 'filtered'
    },
    {
      name: 'no Warehouses',
      args: { ...base, hasWarehouses: false, hasWarehouseController: false },
      expected: 'no-warehouses'
    },
    {
      name: 'Warehouses with no live controller',
      args: { ...base, hasWarehouseController: false },
      expected: 'no-controller'
    },
    {
      name: 'Warehouses that have not discovered anything yet',
      args: base,
      expected: 'no-freight'
    }
  ];

  for (const testCase of testCases) {
    test(testCase.name, () => {
      expect(getVariant(testCase.args)).toBe(testCase.expected);
    });
  }
});
