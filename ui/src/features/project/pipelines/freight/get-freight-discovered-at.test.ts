import { describe, expect, test } from 'vitest';

import { Freight } from '@ui/gen/api/v2/models';

import { getFreightDiscoveredAt } from './get-freight-discovered-at';

const makeFreight = (discoveredAt?: string, creationTimestamp?: string): Freight =>
  ({ discoveredAt, metadata: { creationTimestamp } }) as unknown as Freight;

describe('getFreightDiscoveredAt', () => {
  test('prefers discoveredAt', () => {
    expect(
      getFreightDiscoveredAt(makeFreight('2026-03-10T15:05:19Z', '2026-10-23T09:39:57Z'))
    ).toEqual(new Date('2026-03-10T15:05:19Z'));
  });

  test('falls back to creationTimestamp', () => {
    expect(getFreightDiscoveredAt(makeFreight(undefined, '2026-10-23T09:39:57Z'))).toEqual(
      new Date('2026-10-23T09:39:57Z')
    );
  });

  test('returns null without either', () => {
    expect(getFreightDiscoveredAt(makeFreight())).toBeNull();
    expect(getFreightDiscoveredAt(undefined)).toBeNull();
  });
});
