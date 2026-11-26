import { QueryClient, QueryObserver } from '@tanstack/react-query';
import { describe, expect, test, vi } from 'vitest';

import { readFresh } from './database-reads';

const queryKey = ['statuses'];

// observed sets up a query a page is watching, whose reads each finish only
// when the test says, so a test can tell which read is under way when.
const observed = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const reads: ((value: number) => void)[] = [];
  const observer = new QueryObserver(client, {
    queryKey,
    queryFn: () => new Promise<number>((resolve) => reads.push(resolve))
  });
  const unsubscribe = observer.subscribe(() => {});
  return { client, reads, unsubscribe };
};

describe('readFresh()', () => {
  test('reads again after a first read that was already under way', async () => {
    const { client, reads, unsubscribe } = observed();
    await vi.waitFor(() => expect(reads).toHaveLength(1));

    const done = readFresh(client, queryKey);
    // The read under way may predate the change, so it cannot count as the
    // fresh one; a second starts only once it is done.
    expect(reads).toHaveLength(1);
    reads[0](1);
    await vi.waitFor(() => expect(reads).toHaveLength(2));
    reads[1](2);
    await done;

    expect(client.getQueryData(queryKey)).toBe(2);
    unsubscribe();
  });

  test('replaces a read under way once there is data, rather than reading twice', async () => {
    const { client, reads, unsubscribe } = observed();
    await vi.waitFor(() => expect(reads).toHaveLength(1));
    reads[0](1);
    await vi.waitFor(() => expect(client.getQueryData(queryKey)).toBe(1));

    // A poll is under way when the change comes.
    void client.refetchQueries({ queryKey });
    await vi.waitFor(() => expect(reads).toHaveLength(2));

    const done = readFresh(client, queryKey);
    await vi.waitFor(() => expect(reads).toHaveLength(3));
    reads[2](3);
    await done;

    expect(reads).toHaveLength(3);
    expect(client.getQueryData(queryKey)).toBe(3);
    unsubscribe();
  });

  test('reads once when nothing is under way', async () => {
    const { client, reads, unsubscribe } = observed();
    await vi.waitFor(() => expect(reads).toHaveLength(1));
    reads[0](1);
    await vi.waitFor(() => expect(client.getQueryData(queryKey)).toBe(1));

    const done = readFresh(client, queryKey);
    await vi.waitFor(() => expect(reads).toHaveLength(2));
    reads[1](2);
    await done;

    expect(reads).toHaveLength(2);
    expect(client.getQueryData(queryKey)).toBe(2);
    unsubscribe();
  });
});
