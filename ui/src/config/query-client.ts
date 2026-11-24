import { MutationCache, Query, QueryCache, QueryClient } from '@tanstack/react-query';
import { notification } from 'antd';

import { ApiError } from '@ui/lib/api/custom-fetch';
import { apiErrorBodyMessage } from '@ui/lib/api/error-message';

const showErrorNotification = (error: unknown) => {
  let message: string;
  if (error instanceof ApiError) {
    // Notifications appear on their own, so fall back to ApiError's message
    // ("API Error: 404 Not Found") to keep the status code in view.
    message = apiErrorBodyMessage(error.body) || error.message;
  } else if (error instanceof Error) {
    message = error.message;
  } else {
    message = 'Unexpected API error';
  }
  notification.error({ message, placement: 'bottomRight' });
};

// The queries that opted into notifyErrorOnce and have failed since they last
// succeeded.
const failing = new WeakSet<Query<unknown, unknown, unknown>>();

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Allow queries to opt out of 404 notifications (e.g. ClusterConfig/ProjectConfig
      // which may not exist yet on first setup)
      if (query.meta?.silent404 && error instanceof ApiError && error.isNotFound()) {
        return;
      }
      // Allow queries read on a timer to report a failure once rather than on
      // every tick. The next failure after a successful read is reported anew.
      if (query.meta?.notifyErrorOnce) {
        if (failing.has(query)) {
          return;
        }
        failing.add(query);
      }
      showErrorNotification(error);
    },
    onSuccess: (_data, query) => {
      failing.delete(query);
    }
  }),
  mutationCache: new MutationCache({
    onError: (error) => {
      // REST API errors surface here as ApiError and are shown via a notification
      if (error instanceof ApiError) {
        showErrorNotification(error);
      }
    }
  }),
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      gcTime: 0
    }
  }
});
