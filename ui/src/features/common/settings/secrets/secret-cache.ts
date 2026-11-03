import { QueryClient } from '@tanstack/react-query';

import {
  getListProjectGenericCredentialsQueryKey,
  getListProjectRepoCredentialsQueryKey,
  getListSharedGenericCredentialsQueryKey,
  getListSharedRepoCredentialsQueryKey,
  listProjectGenericCredentialsResponse
} from '@ui/gen/api/v2/credentials/credentials';
import { V1Secret } from '@ui/gen/api/v2/models';

// Keep only the server-returned redacted Secret, never the credential form values.
export const updateCredentialsCache = async (
  client: QueryClient,
  project: string,
  type: 'generic' | 'repo',
  name: string,
  saved?: V1Secret
) => {
  if (
    !name ||
    (saved && (saved.metadata?.name !== name || (project && saved.metadata.namespace !== project)))
  )
    return;
  const key = project
    ? type === 'generic'
      ? getListProjectGenericCredentialsQueryKey(project)
      : getListProjectRepoCredentialsQueryKey(project)
    : type === 'generic'
      ? getListSharedGenericCredentialsQueryKey()
      : getListSharedRepoCredentialsQueryKey();
  await client.cancelQueries({ queryKey: key, exact: true });
  client.setQueryData<listProjectGenericCredentialsResponse>(key, (old) => {
    const items = (old?.data.items || []).filter((item) => item.metadata?.name !== name);
    if (saved) items.push(saved);
    items.sort((a, b) => (a.metadata?.name || '').localeCompare(b.metadata?.name || ''));
    return {
      ...(old || { status: 200 as const, headers: new Headers() }),
      data: { ...old?.data, items }
    };
  });
};
