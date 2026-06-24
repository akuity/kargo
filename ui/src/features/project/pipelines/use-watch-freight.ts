import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetFreightQueryKey,
  getQueryFreightsRestQueryKey,
  getFreightResponse,
  queryFreightsRestResponse
} from '@ui/gen/api/v2/core/core';
import { Freight } from '@ui/gen/api/v2/models';

import { runSeededWatch, upsertOrDelete } from './watch-utils';

// origins must match the params the page's useQueryFreightsRest uses, so the
// seed resourceVersion is read from the same query cache entry. enabled gates
// the watch on the initial list having loaded, so it never opens before a seed
// resourceVersion is available (which would start an unseeded, replaying watch).
export const useWatchFreight = (project: string, origins?: string[], enabled = true) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !enabled) {
      return;
    }

    const abort = new AbortController();
    const listKey = getQueryFreightsRestQueryKey(project, { origins });

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as queryFreightsRestResponse | undefined)?.data
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true' });
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/freight?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, freight: Freight) => {
      // Update all queryFreight caches for this project, including
      // warehouse-filtered variants, which use a different cache key.
      // Using setQueriesData (rather than setQueryData) ensures we only
      // touch caches that already have an active query backing them,
      // avoiding orphaned entries with no queryFn that would crash on
      // refetch.
      client.setQueriesData<queryFreightsRestResponse>(
        { queryKey: getQueryFreightsRestQueryKey(project, undefined), exact: false },
        (old) => {
          if (!old?.data?.groups) {
            return old;
          }
          const updatedGroups = Object.fromEntries(
            Object.entries(old.data.groups).map(([key, group]) => [
              key,
              { ...group, items: upsertOrDelete(group.items ?? [], freight, type) }
            ])
          );
          return { ...old, data: { ...old.data, groups: updatedGroups } };
        }
      );

      const freightKey = getGetFreightQueryKey(project, freight.metadata?.name);

      if (type === 'DELETED') {
        client.removeQueries({ queryKey: freightKey });
      } else {
        client.setQueryData(freightKey, (old: getFreightResponse | undefined) => ({
          ...old,
          data: freight
        }));
      }
    };

    runSeededWatch<Freight>({
      signal: abort.signal,
      buildUrl,
      seedResourceVersion,
      relist,
      onEvent
    });

    return () => abort.abort();
  }, [project, client, enabled, (origins || []).join(',')]);
};
