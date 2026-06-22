import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetWarehouseQueryKey,
  getListWarehousesQueryKey,
  getWarehouseResponse,
  listWarehousesResponse
} from '@ui/gen/api/v2/core/core';
import { Warehouse } from '@ui/gen/api/v2/models';

import { debounce, runSeededWatch, upsertOrDelete } from './watch-utils';

export const useWatchWarehouses = (
  project: string,
  opts?: {
    refreshHook?: () => void;
    onWarehouseEvent?: (warehouse: Warehouse) => void;
  }
) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListWarehousesQueryKey(project);
    const pendingRefresh: Record<string, boolean> = {};

    // Coalesce bursts of warehouse events so the graph recompute is triggered
    // once per burst rather than once per event.
    const emitWarehouseEvent = debounce((warehouse: Warehouse) =>
      opts?.onWarehouseEvent?.(warehouse)
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
              ...old.data,
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

        const refreshRequest = warehouse.metadata?.annotations?.['kargo.akuity.io/refresh'];
        const refreshStatus = warehouse.status?.lastHandledRefresh;
        const isRefreshing = refreshRequest !== undefined && refreshRequest !== refreshStatus;

        if (isRefreshing) {
          pendingRefresh[name] = true;
        } else if (pendingRefresh[name]) {
          delete pendingRefresh[name];
          opts?.refreshHook?.();
        }

        emitWarehouseEvent.call(warehouse);
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
      emitWarehouseEvent.cancel();
    };
  }, [project, client]);
};
