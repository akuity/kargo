import { describe, expect, test } from 'vitest';

import { escapeCancelsAction } from './escape-cancels-action';

const key = (overrides: Partial<KeyboardEvent> = {}) => ({
  key: 'Escape',
  isComposing: false,
  defaultPrevented: false,
  ...overrides
});

const noLayer = { querySelector: () => null };
const openLayer = { querySelector: () => ({}) as Element };

describe('escapeCancelsAction()', () => {
  test('Escape cancels the action', () => {
    expect(escapeCancelsAction(key(), noLayer)).toBe(true);
  });

  test('other keys do not', () => {
    expect(escapeCancelsAction(key({ key: 'Enter' }), noLayer)).toBe(false);
  });

  test('Escape that ends an IME composition does not', () => {
    expect(escapeCancelsAction(key({ isComposing: true }), noLayer)).toBe(false);
  });

  test('Escape something else already handled does not', () => {
    expect(escapeCancelsAction(key({ defaultPrevented: true }), noLayer)).toBe(false);
  });

  test('Escape is left to an open Drawer or Modal', () => {
    expect(escapeCancelsAction(key(), openLayer)).toBe(false);
  });
});
