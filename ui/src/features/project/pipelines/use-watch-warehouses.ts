import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetWarehouseQueryKey,
  getListWarehousesQueryKey,
  getWarehouseResponse,
  listWarehousesResponse
} from '@ui/gen/api/v2/core/core';
import { Warehouse } from '@ui/gen/api/v2/models';

import { readSSEStream, upsertOrDelete } from './watch-utils';

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
    const url = `/v1beta1/projects/${encodeURIComponent(project)}/warehouses?watch=true`;
    const listKey = getListWarehousesQueryKey(project);
    const pendingRefresh: Record<string, boolean> = {};

    (async () => {
      for await (const event of readSSEStream<Warehouse>(url, abort.signal)) {
        const warehouse = event.object;
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
                items: upsertOrDelete(old.data.items ?? [], warehouse, event.type)
              }
            };
          }
        );

        const warehouseKey = getGetWarehouseQueryKey(project, name);

        if (event.type === 'DELETED') {
          client.removeQueries({ queryKey: warehouseKey });
        } else {
          client.setQueriesData(
            {
              exact: false,
              queryKey: warehouseKey
            },
            (old: getWarehouseResponse | undefined) => (old ? { ...old, data: warehouse } : old)
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

          opts?.onWarehouseEvent?.(warehouse);
        }
      }
    })();

    return () => abort.abort();
  }, [project]);
};
