import { createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { transportWithAuth } from '@ui/config/transport';
import { queryCache } from '@ui/features/utils/cache';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { KargoService, QueryFreightResponse } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

const upsertFreight = (
  current: QueryFreightResponse | undefined,
  freight: Freight
): QueryFreightResponse => {
  const existing = current?.groups?.['']?.freight || [];
  const found = existing.some((f) => f?.metadata?.name === freight?.metadata?.name);
  const updated = found
    ? existing.map((f) => (f?.metadata?.name === freight?.metadata?.name ? freight : f))
    : [...existing, freight];
  return {
    ...current,
    groups: {
      ...current?.groups,
      '': { ...current?.groups?.[''], freight: updated }
    }
  } as QueryFreightResponse;
};

const deleteFreight = (
  current: QueryFreightResponse | undefined,
  freight: Freight
): QueryFreightResponse =>
  ({
    ...current,
    groups: {
      ...current?.groups,
      '': {
        ...current?.groups?.[''],
        freight: (current?.groups?.['']?.freight || []).filter(
          (f) => f?.metadata?.name !== freight?.metadata?.name
        )
      }
    }
  }) as QueryFreightResponse;

export const useWatchFreight = (project: string) => {
  const client = useQueryClient();

  useEffect(() => {
    const cancel = new AbortController();

    const watchFreight = async () => {
      const promiseClient = createClient(KargoService, transportWithAuth);

      const stream = promiseClient.watchFreight(
        {
          project
        },
        { signal: cancel.signal }
      );

      for await (const e of stream) {
        const freight = e.freight;

        if (!freight) {
          continue;
        }

        const currentFreight = queryCache.freight.get(project);

        // Skip ADDED events for freight that already exists in the cache.
        // Kubernetes watches replay all existing objects as ADDED on connect,
        // which duplicates the initial GET and causes unnecessary re-renders.
        if (e.type === 'ADDED') {
          const existing = currentFreight?.groups?.['']?.freight || [];
          if (existing.some((f) => f?.metadata?.name === freight?.metadata?.name)) {
            continue;
          }
        }

        if (e.type === 'DELETED') {
          // Remove from all queryFreight caches for this project, including
          // warehouse-filtered variants, which use a different cache key.
          client.setQueriesData<QueryFreightResponse>(
            {
              queryKey: createConnectQueryKey({
                cardinality: 'finite',
                schema: queryFreight,
                input: { project },
                transport: transportWithAuth
              }),
              exact: false
            },
            (current) => deleteFreight(current, freight)
          );
        } else {
          const updatedFreight = upsertFreight(currentFreight, freight);
          const queryFreightKey = createConnectQueryKey({
            cardinality: 'finite',
            schema: queryFreight,
            input: { project },
            transport: transportWithAuth
          });
          client.setQueryData(queryFreightKey, updatedFreight);
        }
      }
    };

    watchFreight();

    return () => cancel.abort();
  }, [project]);
};
