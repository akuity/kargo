import { MutationObserver, QueryClient, QueryObserver } from '@tanstack/react-query';
import { afterEach, describe, expect, test, vi } from 'vitest';

import {
  getGetFreightQueryKey,
  getQueryFreightsRestQueryKey
} from '../../../../gen/api/v2/core/core';
import { Freight } from '../../../../gen/api/v2/models';

import { getApprovalMutationOptions } from './use-approve-freight';

describe('Freight approval', () => {
  afterEach(() => vi.unstubAllGlobals());

  test('refreshes the active Freight list and details for the approved project without a watch event', async () => {
    const client = new QueryClient();
    const project = 'my-project';
    const stage = 'ring-x';
    const freightName = 'freight-id';
    const before: Freight = { metadata: { name: freightName } };
    const after: Freight = {
      ...before,
      status: { approvedFor: { 'ring-x': {} } }
    };
    let freight = before;

    vi.stubGlobal('window', {
      __KARGO_BASE_PATH__: '',
      location: { origin: 'http://localhost', pathname: '/' }
    });
    vi.stubGlobal('localStorage', { getItem: () => null });
    vi.stubGlobal('fetch', async () => {
      freight = after;
      return new Response(null, { status: 200 });
    });

    const list = new QueryObserver(client, {
      queryKey: getQueryFreightsRestQueryKey(project, { origins: ['warehouse'] }),
      queryFn: async () => [freight]
    });
    const detail = new QueryObserver(client, {
      queryKey: getGetFreightQueryKey(project, freightName),
      queryFn: async () => freight
    });
    const other = new QueryObserver(client, {
      queryKey: getQueryFreightsRestQueryKey('other-project'),
      queryFn: async () => [freight]
    });
    const unsubscribe = [
      list.subscribe(() => {}),
      detail.subscribe(() => {}),
      other.subscribe(() => {})
    ];

    try {
      await Promise.all([list.refetch(), detail.refetch(), other.refetch()]);
      const mutation = new MutationObserver(client, getApprovalMutationOptions(client));
      await mutation.mutate({ project, freightNameOrAlias: freightName, params: { stage } });

      expect(list.getCurrentResult().data).toEqual([after]);
      expect(detail.getCurrentResult().data).toEqual(after);
      expect(other.getCurrentResult().data).toEqual([before]);
    } finally {
      unsubscribe.forEach((unsubscribe) => unsubscribe());
      client.clear();
    }
  });
});
