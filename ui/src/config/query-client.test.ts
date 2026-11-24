import { QueryObserver } from '@tanstack/react-query';
import { notification } from 'antd';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { queryClient } from './query-client';

vi.mock('antd', () => ({ notification: { error: vi.fn() } }));

// observe keeps a query in the cache, as a mounted component would, and reads
// it again on demand. Each read answers with the next of the given outcomes: an
// Error to fail with, anything else to succeed with.
const observe = (key: string, meta: Record<string, unknown> | undefined, outcomes: unknown[]) => {
  let read = 0;
  const observer = new QueryObserver(queryClient, {
    queryKey: [key],
    queryFn: async () => {
      const outcome = outcomes[read++];
      if (outcome instanceof Error) {
        throw outcome;
      }
      return outcome;
    },
    meta
  });
  const unsubscribe = observer.subscribe(() => {});
  return { refetch: () => observer.refetch(), unsubscribe };
};

describe('queryClient error notifications', () => {
  const notified = vi.mocked(notification.error);

  beforeEach(() => {
    notified.mockClear();
  });

  afterEach(() => {
    queryClient.clear();
  });

  test('a query reports every failure', async () => {
    const outcomes = [new Error('first'), new Error('second')];
    const { refetch, unsubscribe } = observe('plain', undefined, outcomes);
    await vi.waitFor(() => expect(notified).toHaveBeenCalledTimes(1));
    await refetch();
    expect(notified).toHaveBeenCalledTimes(2);
    unsubscribe();
  });

  test('a notifyErrorOnce query reports a failure once until it succeeds again', async () => {
    const outcomes = [new Error('down'), new Error('down'), 'ok', new Error('down again')];
    const { refetch, unsubscribe } = observe('polled', { notifyErrorOnce: true }, outcomes);
    await vi.waitFor(() => expect(notified).toHaveBeenCalledTimes(1));

    await refetch();
    expect(notified).toHaveBeenCalledTimes(1);

    await refetch();
    await refetch();
    expect(notified).toHaveBeenCalledTimes(2);
    expect(notified).toHaveBeenLastCalledWith(expect.objectContaining({ message: 'down again' }));
    unsubscribe();
  });
});
