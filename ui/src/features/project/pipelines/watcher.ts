import { create } from '@bufbuild/protobuf';
import { Client, createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { QueryClient } from '@tanstack/react-query';

import { transportWithAuth } from '@ui/config/transport';
import { WarehouseExpanded } from '@ui/extend/types';
import { warehouseExpand } from '@ui/extend/warehouse-expand';
import {
  getStage,
  getWarehouse,
  listStages,
  listWarehouses
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import {
  GetStageRequestSchema,
  GetWarehouseRequestSchema,
  KargoService,
  ListStagesRequestSchema,
  ListStagesResponse,
  ListWarehousesRequestSchema,
  ListWarehousesResponse
} from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

// Batch streaming events: accumulate changes and flush the list-level query
// cache update once per animation frame. Per-item callbacks (detail cache,
// graph node updates) still fire immediately so individual views stay fresh.
// Without batching, 400 stage events on page load trigger 400 separate
// Pipelines re-renders, each recomputing O(n) derived hooks.
async function ProcessEvents<T extends { type: string }, S extends { metadata?: ObjectMeta }>(
  stream: AsyncIterable<T>,
  getData: () => S[],
  getter: (e: T) => S,
  onFlush: (data: S[]) => void,
  onItem?: (item: S) => void
) {
  let data = getData();
  let dirty = false;
  let rafId = 0;

  const flush = () => {
    rafId = 0;
    if (!dirty) return;
    dirty = false;
    onFlush(data);
  };

  const scheduleFlush = () => {
    if (!rafId) {
      rafId = requestAnimationFrame(flush);
    }
  };

  for await (const e of stream) {
    // Read latest cache on first event of a batch
    if (!dirty) {
      data = getData();
    }

    const item = getter(e);
    const index = data.findIndex((d) => d.metadata?.name === item.metadata?.name);

    if (e.type === 'DELETED') {
      if (index !== -1) {
        data = [...data.slice(0, index), ...data.slice(index + 1)];
      }
    } else {
      if (index === -1) {
        data = [...data, item];
      } else {
        data = [...data.slice(0, index), item, ...data.slice(index + 1)];
      }
    }

    dirty = true;
    onItem?.(item);
    scheduleFlush();
  }

  // Flush remaining if stream ends
  if (rafId) {
    cancelAnimationFrame(rafId);
  }
  if (dirty) {
    onFlush(data);
  }
}

export class Watcher {
  cancel: AbortController;
  client: QueryClient;
  promiseClient: Client<typeof KargoService>;
  project: string;

  constructor(project: string, client: QueryClient) {
    this.cancel = new AbortController();
    this.client = client;
    this.project = project;
    this.promiseClient = createClient(KargoService, transportWithAuth);
  }

  cancelWatch() {
    this.cancel.abort();
  }

  async watchStages(
    // utilise the fact that something changed in this stage
    // avoid as much as re-construction of data as possible by using this parameter
    onStageEvent?: (stage: Stage) => void
  ) {
    const stream = this.promiseClient.watchStages(
      { project: this.project },
      { signal: this.cancel.signal }
    );

    ProcessEvents(
      stream,
      () => {
        const data = this.client.getQueryData(
          createConnectQueryKey({
            schema: listStages,
            input: create(ListStagesRequestSchema, { project: this.project }),
            cardinality: 'finite',
            transport: transportWithAuth
          })
        );

        return (data as ListStagesResponse)?.stages || [];
      },
      (e) => e.stage as Stage,
      (data) => {
        // Batched: update stages list once per animation frame
        const listStagesQueryKey = createConnectQueryKey({
          schema: listStages,
          input: create(ListStagesRequestSchema, { project: this.project }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(listStagesQueryKey, {
          stages: data,
          $typeName: 'akuity.io.kargo.service.v1alpha1.ListStagesResponse'
        });
      },
      (stage) => {
        // Per-item: update individual stage detail cache + notify graph
        const getStageQueryKey = createConnectQueryKey({
          schema: getStage,
          input: create(GetStageRequestSchema, {
            project: this.project,
            name: stage.metadata?.name
          }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(getStageQueryKey, {
          result: {
            value: stage,
            case: 'stage'
          },
          $typeName: 'akuity.io.kargo.service.v1alpha1.GetStageResponse'
        });

        onStageEvent?.(stage);
      }
    );
  }

  async watchWarehouses(opts?: {
    refreshHook?: () => void;
    onWarehouseEvent?: (warehouse: WarehouseExpanded) => void;
  }) {
    const stream = this.promiseClient.watchWarehouses(
      { project: this.project },
      { signal: this.cancel.signal }
    );
    const refresh = {} as { [key: string]: boolean };

    ProcessEvents(
      stream,
      () => {
        const data = this.client.getQueryData(
          createConnectQueryKey({
            schema: listWarehouses,
            input: create(ListWarehousesRequestSchema, { project: this.project }),
            cardinality: 'finite',
            transport: transportWithAuth
          })
        );

        return (data as ListWarehousesResponse)?.warehouses?.map((w) => warehouseExpand(w)) || [];
      },
      (e) => e.warehouse as WarehouseExpanded,
      (data) => {
        // Batched: update warehouses list once per animation frame
        const listWarehousesQueryKey = createConnectQueryKey({
          schema: listWarehouses,
          input: create(ListWarehousesRequestSchema, {
            project: this.project
          }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(listWarehousesQueryKey, {
          // @ts-expect-error warehouse expanded
          warehouses: data,
          $typeName: 'akuity.io.kargo.service.v1alpha1.ListWarehousesResponse'
        });
      },
      (warehouse) => {
        // Per-item: refresh logic + detail cache + notify graph
        const refreshRequest = warehouse?.metadata?.annotations['kargo.akuity.io/refresh'];
        const refreshStatus = warehouse?.status?.lastHandledRefresh;
        const refreshing = refreshRequest !== undefined && refreshRequest !== refreshStatus;
        if (refreshing) {
          refresh[warehouse?.metadata?.name || ''] = true;
        } else if (refresh[warehouse?.metadata?.name || '']) {
          delete refresh[warehouse?.metadata?.name || ''];
          opts?.refreshHook?.();
        }

        const getWarehouseQueryKey = createConnectQueryKey({
          schema: getWarehouse,
          input: create(GetWarehouseRequestSchema, {
            project: this.project,
            name: warehouse.metadata?.name
          }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(getWarehouseQueryKey, {
          $typeName: 'akuity.io.kargo.service.v1alpha1.GetWarehouseResponse',
          result: {
            // @ts-expect-error warehouse expanded
            value: warehouse,
            case: 'warehouse'
          }
        });

        opts?.onWarehouseEvent?.(warehouse);
      }
    );
  }
}
