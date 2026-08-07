import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { batchEmitter } from './watch-utils';

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

  test('cancel drops the pending batch without flushing', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    emitter.cancel();

    vi.runAllTimers();

    expect(flush).not.toHaveBeenCalled();
  });

  test('cancel clears accumulated items so a later call starts fresh', () => {
    const flush = vi.fn<(items: Obj[]) => void>();
    const emitter = batchEmitter(flush, keyOf);

    emitter.call({ name: 'a', value: 1 });
    emitter.cancel();

    emitter.call({ name: 'b', value: 2 });
    vi.runAllTimers();

    expect(flush).toHaveBeenCalledTimes(1);
    expect(flush.mock.calls[0][0]).toEqual([{ name: 'b', value: 2 }]);
  });
});
