import { QueryClient } from '@tanstack/react-query';

import {
  getListSystemGenericCredentialsQueryKey,
  listSystemGenericCredentialsResponse
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';

// A successful write is newer than a LIST served by the Kubernetes informer.
// Apply the server's redacted response instead of immediately replacing it with
// an eventually consistent refetch. Cancel older in-flight reads as well.
export const updateSystemSecretCache = async (
  client: QueryClient,
  name: string,
  saved?: V1Secret
) => {
  const queryKey = getListSystemGenericCredentialsQueryKey();
  await client.cancelQueries({ queryKey, exact: true });
  client.setQueryData<listSystemGenericCredentialsResponse>(queryKey, (old) => {
    const items = (old?.data.items || []).filter((item) => item.metadata?.name !== name);
    if (saved) items.push(saved);
    items.sort((a, b) => (a.metadata?.name || '').localeCompare(b.metadata?.name || ''));
    return {
      ...(old || { status: 200 as const, headers: new Headers() }),
      data: { ...old?.data, items }
    };
  });
};
