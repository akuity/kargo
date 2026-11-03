import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import {
  getListProjectGenericCredentialsQueryKey,
  getListProjectRepoCredentialsQueryKey,
  getListSharedGenericCredentialsQueryKey,
  getListSharedRepoCredentialsQueryKey
} from '../../../../gen/api/v2/credentials/credentials';
import { V1Secret } from '../../../../gen/api/v2/models';

import { updateCredentialsCache } from './secret-cache';

const saved: V1Secret = {
  metadata: { name: 'owned-secret', namespace: 'shared' },
  data: { key: 'redacted' }
};
describe('confirmed credential list cache', () => {
  it('keeps the server-returned redacted Secret when an older empty list finishes', async () => {
    const client = new QueryClient();
    const key = getListSharedGenericCredentialsQueryKey();
    let finish!: (value: unknown) => void;
    const pending = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await updateCredentialsCache(client, '', 'generic', 'owned-secret', saved);
    finish({ data: { items: [] } });
    await pending;
    expect(client.getQueryData<{ data: { items: V1Secret[] } }>(key)?.data.items).toEqual([saved]);
    client.clear();
  });
  it('isolates generic, repository, shared and project caches while applying edits and deletions', async () => {
    const client = new QueryClient();
    const keys = [
      getListSharedGenericCredentialsQueryKey(),
      getListSharedRepoCredentialsQueryKey(),
      getListProjectGenericCredentialsQueryKey('project'),
      getListProjectRepoCredentialsQueryKey('project')
    ];
    for (const key of keys) client.setQueryData(key, { data: { items: [saved] } });
    const edited = {
      ...saved,
      metadata: { ...saved.metadata, namespace: 'project' },
      type: 'Opaque'
    };
    await updateCredentialsCache(client, 'project', 'repo', 'owned-secret', edited);
    expect(client.getQueryData<{ data: { items: V1Secret[] } }>(keys[3])?.data.items).toEqual([
      edited
    ]);
    await updateCredentialsCache(client, 'project', 'repo', 'owned-secret');
    expect(client.getQueryData<{ data: { items: V1Secret[] } }>(keys[3])?.data.items).toEqual([]);
    for (const key of keys.slice(0, 3))
      expect(client.getQueryData(key)).toEqual({ data: { items: [saved] } });
    client.clear();
  });
  it('rejects foreign namespaces and mismatched names', async () => {
    const client = new QueryClient();
    await updateCredentialsCache(client, 'project', 'generic', 'owned-secret', saved);
    await updateCredentialsCache(client, '', 'generic', 'different', saved);
    expect(
      client.getQueryData(getListProjectGenericCredentialsQueryKey('project'))
    ).toBeUndefined();
    expect(client.getQueryData(getListSharedGenericCredentialsQueryKey())).toBeUndefined();
    client.clear();
  });
});
