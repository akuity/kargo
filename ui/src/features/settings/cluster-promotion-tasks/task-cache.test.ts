import { QueryClient } from '@tanstack/react-query';
import { describe, expect, it } from 'vitest';

import {
  getListClusterPromotionTasksQueryKey,
  listClusterPromotionTasksResponse
} from '../../../gen/api/v2/core/core';

import { updateClusterPromotionTaskCache } from './task-cache';

const key = getListClusterPromotionTasksQueryKey();
const task = {
  apiVersion: 'kargo.akuity.io/v1alpha1',
  kind: 'ClusterPromotionTask',
  metadata: { name: 'owned', uid: 'uid' },
  spec: { steps: [{ uses: 'http', config: { url: 'https://example.invalid' } }] }
};
const response = (items: (typeof task)[]): listClusterPromotionTasksResponse => ({
  data: { items },
  status: 200,
  headers: new Headers()
});

describe('ClusterPromotionTask mutation cache', () => {
  it('keeps a confirmed creation when an older empty list finishes late', async () => {
    const client = new QueryClient();
    let finish!: (value: listClusterPromotionTasksResponse) => void;
    const query = client
      .fetchQuery({
        queryKey: key,
        queryFn: () =>
          new Promise<listClusterPromotionTasksResponse>((resolve) => {
            finish = resolve;
          })
      })
      .catch(() => undefined);
    await updateClusterPromotionTaskCache(client, [{ createdResourceManifest: task }]);
    finish(response([]));
    await query;
    expect(client.getQueryData<listClusterPromotionTasksResponse>(key)?.data.items).toEqual([task]);
    client.clear();
  });

  it('applies successful edits and deletions without replacing unrelated tasks', async () => {
    const client = new QueryClient();
    const other = { ...task, metadata: { name: 'other', uid: 'other-uid' } };
    client.setQueryData(key, response([task, other]));
    const edited = {
      ...task,
      spec: { ...task.spec, vars: [{ name: 'message', value: 'updated' }] }
    };
    await updateClusterPromotionTaskCache(client, [{ updatedResourceManifest: edited }]);
    expect(client.getQueryData<listClusterPromotionTasksResponse>(key)?.data.items).toEqual([
      other,
      edited
    ]);
    await updateClusterPromotionTaskCache(client, [{ deletedResourceManifest: edited }]);
    expect(client.getQueryData<listClusterPromotionTasksResponse>(key)?.data.items).toEqual([
      other
    ]);
    client.clear();
  });

  it('does not apply failed writes, other resource kinds or malformed results', async () => {
    const client = new QueryClient();
    client.setQueryData(key, response([task]));
    await updateClusterPromotionTaskCache(client, [
      { error: 'denied', deletedResourceManifest: task },
      { createdResourceManifest: { ...task, kind: 'PromotionTask' } },
      { createdResourceManifest: { kind: 'ClusterPromotionTask' } }
    ]);
    expect(client.getQueryData<listClusterPromotionTasksResponse>(key)?.data.items).toEqual([task]);
    client.clear();
  });
});
