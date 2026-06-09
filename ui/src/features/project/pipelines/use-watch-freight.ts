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

        client.setQueryData(listKey, (old: queryFreightsRestResponse | undefined) => {
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
        });

        const freightKey = getGetFreightQueryKey(project, freight.metadata?.name);

        if (event.type === 'DELETED') {
          client.removeQueries({ queryKey: freightKey });
        } else {
          client.setQueryData(freightKey, (old: getFreightResponse | undefined) =>
            old ? { ...old, data: freight } : old
          );
        }
      }
    })();

    return () => abort.abort();
  }, [project]);
};
