import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetWarehouseQueryKey,
  getListWarehousesQueryKey,
  getWarehouseResponse,
  listWarehousesResponse
} from '@ui/gen/api/v2/core/core';
import { Warehouse } from '@ui/gen/api/v2/models';

import {
  advanceResourceVersion,
  batchEmitter,
  runSeededWatch,
  upsertOrDelete
} from './watch-utils';

// A Warehouse is refreshing while it has a refresh request it has not handled.
const isRefreshing = (warehouse: Warehouse) => {
  const refreshRequest = warehouse.metadata?.annotations?.['kargo.akuity.io/refresh'];
  return refreshRequest !== undefined && refreshRequest !== warehouse.status?.lastHandledRefresh;
};

// pendingRefreshes marks the Warehouses already refreshing when a watch opens.
// A reopened watch resumes after the last event it saw, so it never redelivers
// the event that started a refresh; without this, the one that ends it would
// not be recognized as a completion.
export const pendingRefreshes = (warehouses: Warehouse[]): Record<string, boolean> =>
  Object.fromEntries(
    warehouses.filter(isRefreshing).map((warehouse) => [warehouse.metadata?.name || '', true])
  );

export const useWatchWarehouses = (
  project: string,
  opts?: {
    refreshHook?: () => void;
    onWarehousesEvent?: (warehouses: Warehouse[]) => void;
  }
) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListWarehousesQueryKey(project);
    const pendingRefresh = pendingRefreshes(
      (client.getQueryData(listKey) as listWarehousesResponse | undefined)?.data?.items ?? []
    );

    // Batch bursts of warehouse events so the graph applies them all at once
    // rather than once per event. Keying by name keeps every distinct
    // warehouse's latest update — a plain debounce would drop all but the last
    // object in a burst.
    const emitWarehouseEvents = batchEmitter(
      (warehouses: Warehouse[]) => opts?.onWarehousesEvent?.(warehouses),
      (warehouse) => warehouse.metadata?.name ?? ''
    );

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listWarehousesResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true' });
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/warehouses?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, warehouse: Warehouse) => {
      const name = warehouse.metadata?.name || '';

      client.setQueriesData(
        { exact: false, queryKey: listKey },
        (old: listWarehousesResponse | undefined) => {
          if (!old?.data) {
            return old;
          }
          return {
            ...old,
            data: {
              ...advanceResourceVersion(old.data, warehouse),
              items: upsertOrDelete(old.data.items ?? [], warehouse, type)
            }
          };
        }
      );

      const warehouseKey = getGetWarehouseQueryKey(project, name);

      if (type === 'DELETED') {
        client.removeQueries({ queryKey: warehouseKey });
      } else {
        client.setQueriesData(
          {
            exact: false,
            queryKey: warehouseKey
          },
          (old: getWarehouseResponse | undefined) =>
            old
              ? {
                  ...old,
                  data: {
                    // WATCH ENDPOINT WAREHOUSE COMES WITHOUT kind, apiVersion etc..
                    // SO WE NEED TO PRESERVE IT FROM INITIAL DATA
                    ...old?.data,
                    ...warehouse
                  }
                }
              : old
        );

        if (isRefreshing(warehouse)) {
          pendingRefresh[name] = true;
        } else if (pendingRefresh[name]) {
          delete pendingRefresh[name];
          opts?.refreshHook?.();
        }

        emitWarehouseEvents.call(warehouse);
      }
    };

    runSeededWatch<Warehouse>({
      signal: abort.signal,
      buildUrl,
      seedResourceVersion,
      relist,
      onEvent
    });

    return () => {
      abort.abort();
      emitWarehouseEvents.flush();
    };
  }, [project, client]);
};
