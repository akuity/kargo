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
