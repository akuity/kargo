import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import {
  getListSystemGenericCredentialsQueryKey,
  listSystemGenericCredentialsResponse
} from '../../../gen/api/v2/credentials/credentials';

import { updateSystemSecretCache } from './secret-cache';

const key = getListSystemGenericCredentialsQueryKey();
const saved = { metadata: { name: 'owned', uid: 'uid' }, stringData: { token: 'REDACTED' } };
const response = (items: (typeof saved)[]): listSystemGenericCredentialsResponse => ({
  data: { items },
  status: 200,
  headers: new Headers()
});

describe('system Secret mutation cache', () => {
  it('retains the confirmed Secret when an older empty list finishes late', async () => {
    const client = new QueryClient();
    let finish!: (value: listSystemGenericCredentialsResponse) => void;
    const query = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise<listSystemGenericCredentialsResponse>((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await updateSystemSecretCache(client, 'owned', saved);
    finish(response([]));
    await query;
    expect(client.getQueryData<listSystemGenericCredentialsResponse>(key)?.data.items).toEqual([
      saved
    ]);
    client.clear();
  });

  it('updates only the named row and removes it after successful deletion', async () => {
    const client = new QueryClient();
    const other = { ...saved, metadata: { name: 'other', uid: 'other-uid' } };
    client.setQueryData(key, response([saved, other]));
    const edited = { ...saved, stringData: { replacement: 'REDACTED' } };
    await updateSystemSecretCache(client, 'owned', edited);
    expect(client.getQueryData<listSystemGenericCredentialsResponse>(key)?.data.items).toEqual([
      other,
      edited
    ]);
    await updateSystemSecretCache(client, 'owned');
    expect(client.getQueryData<listSystemGenericCredentialsResponse>(key)?.data.items).toEqual([
      other
    ]);
    client.clear();
  });
});
