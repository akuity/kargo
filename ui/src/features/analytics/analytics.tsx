import { useEffect } from 'react';

import { useGetPublicConfig } from '@ui/gen/api/v2/system/system';

import { startPostHog } from './posthog';

/**
 * Analytics starts PostHog when, and only when, the API server says an
 * operator has enabled it. It renders nothing.
 */
export const Analytics = () => {
  const { data: response } = useGetPublicConfig();
  const { apiKey, host } = response?.data?.posthogConfig ?? {};

  useEffect(() => {
    if (!apiKey || !host) {
      return;
    }
    // Analytics are best effort: failing to load them must not disturb the UI.
    startPostHog(apiKey, host).catch(() => undefined);
  }, [apiKey, host]);

  return null;
};
