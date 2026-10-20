import { QueryClient } from '@tanstack/react-query';
import { describe, expect, test, vi } from 'vitest';

import { shareWatch } from './use-watch-promotion-requests';

describe('shareWatch', () => {
  test('opens one watch per key however many subscribe, and closes it after the last', () => {
    const client = new QueryClient();
    const signals: AbortSignal[] = [];
    const start = vi.fn((signal: AbortSignal) => {
      signals.push(signal);
    });

    const first = shareWatch(client, 'a', start);
    const second = shareWatch(client, 'a', start);
    expect(start).toHaveBeenCalledTimes(1);

    first();
    expect(signals[0].aborted).toBe(false);
    second();
    expect(signals[0].aborted).toBe(true);

    // A later subscriber opens a fresh watch.
    const third = shareWatch(client, 'a', start);
    expect(start).toHaveBeenCalledTimes(2);
    expect(signals[1].aborted).toBe(false);
    third();
  });

  test('keeps watches of different keys and query clients apart', () => {
    const client = new QueryClient();
    const other = new QueryClient();
    const start = vi.fn();

    const unsubscribers = [
      shareWatch(client, 'a', start),
      shareWatch(client, 'b', start),
      shareWatch(other, 'a', start)
    ];
    expect(start).toHaveBeenCalledTimes(3);
    unsubscribers.forEach((unsubscribe) => unsubscribe());
  });
});
