import { describe, expect, test } from 'vitest';

import { minDiscoveringMs } from '../nodes/warehouse-state';

import { holdDiscovering } from './use-held-discovering';

describe('holdDiscovering()', () => {
  const start = 1_000_000;

  test('a Warehouse discovering now is held, from when it was first seen', () => {
    const startedAt: Record<string, number> = {};
    const { held, releaseAt } = holdDiscovering(['nginx'], startedAt, start);

    expect([...held]).toEqual(['nginx']);
    expect(startedAt).toEqual({ nginx: start });
    expect(releaseAt).toBeUndefined();
  });

  test('a long discovery keeps the start it was first seen at', () => {
    const startedAt = { nginx: start };
    holdDiscovering(['nginx'], startedAt, start + 5000);

    expect(startedAt).toEqual({ nginx: start });
  });

  test('a discovery that ends early is held until its time is up', () => {
    const { held, releaseAt } = holdDiscovering([], { nginx: start }, start + 300);

    expect([...held]).toEqual(['nginx']);
    expect(releaseAt).toBe(start + minDiscoveringMs);
  });

  test('once its time is up it is released and its start forgotten', () => {
    const startedAt = { nginx: start };
    const { held, releaseAt } = holdDiscovering([], startedAt, start + minDiscoveringMs);

    expect(held.size).toBe(0);
    expect(startedAt).toEqual({});
    expect(releaseAt).toBeUndefined();
  });

  test('the earliest of several holds is the one to wake for', () => {
    const { releaseAt } = holdDiscovering([], { nginx: start + 200, redis: start }, start + 500);

    expect(releaseAt).toBe(start + minDiscoveringMs);
  });
});
