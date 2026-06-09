import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetFreightQueryKey,
  getQueryFreightsRestQueryKey,
  getFreightResponse,
  queryFreightsRestResponse
} from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

import { readSSEStream, upsertOrDelete } from './watch-utils';

export const useWatchFreight = (project: string) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const abort = new AbortController();
    const url = `/v1beta1/projects/${encodeURIComponent(project)}/freight?watch=true`;
    const listKey = getQueryFreightsRestQueryKey(project, undefined);

    (async () => {
      for await (const event of readSSEStream<Freight>(url, abort.signal)) {
        const freight = event.object;

        // Update all queryFreight caches for this project, including
        // warehouse-filtered variants, which use a different cache key.
        // Using setQueriesData (rather than setQueryData) ensures we only
        // touch caches that already have an active query backing them,
        // avoiding orphaned entries with no queryFn that would crash on
        // refetch.
        client.setQueriesData<queryFreightsRestResponse>(
          { queryKey: listKey, exact: false },
          (old) => {
            if (!old?.data?.groups) {
              return old;
            }
            const updatedGroups = Object.fromEntries(
              Object.entries(old.data.groups).map(([key, group]) => [
                key,
                { ...group, items: upsertOrDelete(group.items ?? [], freight, event.type) }
              ])
            );
            return { ...old, data: { ...old.data, groups: updatedGroups } };
          }
        );

        const freightKey = getGetFreightQueryKey(project, freight.metadata?.name);

        if (event.type === 'DELETED') {
          client.removeQueries({ queryKey: freightKey });
        } else {
          client.setQueryData(freightKey, (old: getFreightResponse | undefined) => ({
            ...old,
            data: freight
          }));
        }
      }
    })();

    return () => abort.abort();
  }, [project]);
};
