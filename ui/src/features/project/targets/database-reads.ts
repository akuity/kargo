import { QueryClient, QueryKey } from '@tanstack/react-query';

import { ApiError } from '@ui/lib/api/custom-fetch';

const notImplemented = 501;

// polled is the meta of a list read on a timer. Such a list fails on every
// tick for as long as whatever broke it lasts, and reports that once, not
// once per tick.
export const polled = { notifyErrorOnce: true };

// orEmpty answers a read the server cannot serve for want of a database. Such
// a server keeps no Targets at all, so to a reader its fleet is empty, not
// broken.
export const orEmpty = async <T>(read: () => Promise<T>, empty: T): Promise<T> => {
  try {
    return await read();
  } catch (error) {
    if (error instanceof ApiError && error.is(notImplemented)) {
      return empty;
    }
    throw error;
  }
};

// readFresh reads a query again with a read that starts now, for when
// something the query depends on has just changed. A read already under way
// with data before it is dropped for a new one. One under way with nothing
// read yet is not: TanStack hands it back instead of starting another, though
// it may have started before the change. So it is let finish, and read after.
export const readFresh = async (client: QueryClient, queryKey: QueryKey): Promise<void> => {
  const state = client.getQueryState(queryKey);
  const joined = state?.fetchStatus === 'fetching' && state.data === undefined;
  await client.refetchQueries({ queryKey, exact: true });
  if (joined) {
    await client.refetchQueries({ queryKey, exact: true });
  }
};
