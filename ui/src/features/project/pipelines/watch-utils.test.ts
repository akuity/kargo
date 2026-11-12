import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { advanceResourceVersion, batchEmitter } from './watch-utils';

type Obj = { name: string; value: number };

const keyOf = (o: Obj) => o.name;

describe('batchEmitter', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test('delivers every distinct object in a synchronous burst as one batch', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    emitter.call({ name: 'b', value: 2 });
    emitter.call({ name: 'c', value: 3 });

    expect(flush).not.toHaveBeenCalled();

    vi.runAllTimers();

    expect(flush).toHaveBeenCalledTimes(1);
    expect(flush.mock.calls[0][0]).toEqual([
      { name: 'a', value: 1 },
      { name: 'b', value: 2 },
      { name: 'c', value: 3 }
    ]);
  });

  test('collapses repeated updates to the same key to the latest value', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    emitter.call({ name: 'b', value: 1 });
    emitter.call({ name: 'a', value: 2 });
    emitter.call({ name: 'a', value: 3 });

    vi.runAllTimers();

    expect(flush).toHaveBeenCalledTimes(1);
    expect(flush.mock.calls[0][0]).toEqual([
      { name: 'a', value: 3 },
      { name: 'b', value: 1 }
    ]);
  });

  test('resets pending state after each flush', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    vi.runAllTimers();

    emitter.call({ name: 'b', value: 2 });
    vi.runAllTimers();

    expect(flush).toHaveBeenCalledTimes(2);
    expect(flush.mock.calls[0][0]).toEqual([{ name: 'a', value: 1 }]);
    expect(flush.mock.calls[1][0]).toEqual([{ name: 'b', value: 2 }]);
  });

  test('flush delivers the pending batch immediately and only once', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    emitter.call({ name: 'b', value: 2 });
    emitter.flush();

    expect(flush).toHaveBeenCalledTimes(1);
    expect(flush.mock.calls[0][0]).toEqual([
      { name: 'a', value: 1 },
      { name: 'b', value: 2 }
    ]);

    vi.runAllTimers();

    expect(flush).toHaveBeenCalledTimes(1);
  });

  test('flush does nothing when no batch is pending', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.flush();

    emitter.call({ name: 'a', value: 1 });
    vi.runAllTimers();
    emitter.flush();

    expect(flush).toHaveBeenCalledTimes(1);
    expect(flush.mock.calls[0][0]).toEqual([{ name: 'a', value: 1 }]);
  });
});

describe('advanceResourceVersion', () => {
  type List = { metadata?: { resourceVersion?: string; continue?: string }; items: string[] };

  test('moves the list resourceVersion up to a newer event object', () => {
    const list: List = { metadata: { resourceVersion: '100', continue: 'x' }, items: ['a'] };

    expect(advanceResourceVersion(list, { metadata: { resourceVersion: '150' } })).toEqual({
      metadata: { resourceVersion: '150', continue: 'x' },
      items: ['a']
    });
  });

  test('keeps the list resourceVersion for an older or equal event object', () => {
    const list: List = { metadata: { resourceVersion: '100' }, items: [] };

    expect(advanceResourceVersion(list, { metadata: { resourceVersion: '90' } })).toBe(list);
    expect(advanceResourceVersion(list, { metadata: { resourceVersion: '100' } })).toBe(list);
  });

  test('keeps the list resourceVersion when the event object has none', () => {
    const list: List = { metadata: { resourceVersion: '100' }, items: [] };

    expect(advanceResourceVersion(list, {})).toBe(list);
  });

  test('sets a resourceVersion on a list that has none', () => {
    const list: List = { items: ['a'] };

    expect(advanceResourceVersion(list, { metadata: { resourceVersion: '5' } })).toEqual({
      metadata: { resourceVersion: '5' },
      items: ['a']
    });
  });
});
