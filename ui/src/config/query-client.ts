import { MutationCache, QueryCache, QueryClient } from '@tanstack/react-query';
import { notification } from 'antd';

import { ApiError } from '@ui/lib/api/custom-fetch';

const showErrorNotification = (error: unknown) => {
  let message: string;
  if (error instanceof ApiError) {
    const body = error.body;
    if (typeof body === 'object' && body !== null) {
      const b = body as Record<string, unknown>;
      message = String(b.message ?? b.error ?? error.message);
    } else {
      message = error.message;
    }
  } else if (error instanceof Error) {
    message = error.message;
  } else {
    message = 'Unexpected API error';
  }
  notification.error({ message, placement: 'bottomRight' });
};

export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => {
      // Allow queries to opt out of 404 notifications (e.g. ClusterConfig/ProjectConfig
      // which may not exist yet on first setup)
      if (query.meta?.silent404 && error instanceof ApiError && error.isNotFound()) {
        return;
      }
      showErrorNotification(error);
    }
  }),
  mutationCache: new MutationCache({
    onError: (error) => {
      // ConnectRPC errors are handled by the transport interceptor (transport.ts)
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
