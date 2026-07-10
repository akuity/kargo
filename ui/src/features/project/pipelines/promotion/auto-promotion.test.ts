import { describe, expect, test } from 'vitest';

import type { AutoPromotionHold, Stage } from '@ui/gen/api/v2/models';

import {
  getAutoPromotionHold,
  getAutoPromotionHoldEntries,
  holdStateMessage,
  originKey
} from './auto-promotion';

const stageWithHolds = (holds: Record<string, AutoPromotionHold>): Stage =>
  ({
    status: { autoPromotionHolds: holds }
  }) as Stage;

describe('auto-promotion helpers', () => {
  test('originKey requires kind and name', () => {
    expect(originKey()).toBe('');
    expect(originKey({ kind: 'Warehouse' })).toBe('');
    expect(originKey({ kind: 'Warehouse', name: 'warehouse-a' })).toBe('Warehouse/warehouse-a');
  });

  test('getAutoPromotionHold looks up hold by origin', () => {
    const hold = { freightName: 'freight-a' };
    const stage = stageWithHolds({ 'Warehouse/warehouse-a': hold });

    expect(getAutoPromotionHold(stage, { kind: 'Warehouse', name: 'warehouse-a' })).toBe(hold);
    expect(getAutoPromotionHold(stage, { kind: 'Warehouse', name: 'warehouse-b' })).toBeUndefined();
  });

  test('getAutoPromotionHoldEntries sorts by origin key', () => {
    const stage = stageWithHolds({
      'Warehouse/warehouse-b': { freightName: 'freight-b' },
      'Warehouse/warehouse-a': { freightName: 'freight-a' }
    });

    expect(getAutoPromotionHoldEntries(stage).map((entry) => entry.key)).toEqual([
      'Warehouse/warehouse-a',
      'Warehouse/warehouse-b'
    ]);
  });

  test('holdStateMessage includes one or all held origins', () => {
    const stage = stageWithHolds({
      'Warehouse/warehouse-b': {},
      'Warehouse/warehouse-a': {}
    });

    expect(holdStateMessage(stage, { kind: 'Warehouse', name: 'warehouse-a' })).toBe(
      'Auto-promotion paused: Warehouse/warehouse-a'
    );
    expect(holdStateMessage(stage)).toBe(
      'Auto-promotion paused: Warehouse/warehouse-a, Warehouse/warehouse-b'
    );
  });
});
