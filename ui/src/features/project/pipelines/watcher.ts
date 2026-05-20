import { create } from '@bufbuild/protobuf';
import { Client, createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { QueryClient } from '@tanstack/react-query';

import { transportWithAuth } from '@ui/config/transport';
import {
  isExpiredResourceVersionError,
  isSameOrOlderResourceVersion
} from '@ui/features/utils/resource-version';
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
import { Stage, Warehouse } from '@ui/gen/api/v1alpha1/generated_pb';
import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

type ProcessEventsResult = 'completed' | 'resourceVersionExpired';

async function ProcessEvents<T extends { type: string }, S extends { metadata?: ObjectMeta }>(
  stream: AsyncIterable<T>,
  getData: () => S[],
  getter: (e: T) => S,
  callback: (item: S, data: S[]) => void
): Promise<ProcessEventsResult> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    for await (const e of stream) {
      const eventObject = getter(e);
      let data = getData();
      const index = data.findIndex((item) => item.metadata?.name === eventObject.metadata?.name);
      if (e.type === 'DELETED') {
        if (index !== -1) {
          data = [...data.slice(0, index), ...data.slice(index + 1)];
        }
      } else if (
        e.type === 'ADDED' &&
        index !== -1 &&
        isSameOrOlderResourceVersion(data[index], eventObject)
      ) {
        continue;
      } else {
        if (index === -1) {
          data = [...data, eventObject];
        } else {
          data = [...data.slice(0, index), eventObject, ...data.slice(index + 1)];
        }
      }

      clearTimeout(timer);
      timer = setTimeout(() => callback(eventObject, data));
    }
  } catch (err) {
    if (isExpiredResourceVersionError(err)) {
      return 'resourceVersionExpired';
    }
    throw err;
  }
  return 'completed';
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
    onStageEvent?: (stage: Stage) => void,
    warehouses?: string[]
  ) {
    const stagesInput = create(ListStagesRequestSchema, {
      project: this.project,
      freightOrigins: warehouses || []
    });
    const listStagesQueryKey = createConnectQueryKey({
      schema: listStages,
      input: stagesInput,
      cardinality: 'finite',
      transport: transportWithAuth
    });

    try {
      while (!this.cancel.signal.aborted) {
        const stagesResponse = this.client.getQueryData(listStagesQueryKey) as
          | ListStagesResponse
          | undefined;
        const stream = this.promiseClient.watchStages(
          {
            project: this.project,
            freightOrigins: warehouses || [],
            resourceVersion: stagesResponse?.resourceVersion || ''
          },
          { signal: this.cancel.signal }
        );

        const result = await ProcessEvents(
          stream,
          () => {
            const data = this.client.getQueryData(listStagesQueryKey);

            return (data as ListStagesResponse)?.stages || [];
          },
          (e) => e.stage as Stage,
          (stage, data) => {
            // update Stages list
            this.client.setQueryData(listStagesQueryKey, {
              stages: data,
              resourceVersion:
                stage.metadata?.resourceVersion ||
                (this.client.getQueryData(listStagesQueryKey) as ListStagesResponse)
                  ?.resourceVersion ||
                '',
              $typeName: 'akuity.io.kargo.service.v1alpha1.ListStagesResponse'
            });

            // update Stage details
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
        if (result !== 'resourceVersionExpired') {
          return;
        }
        await this.client.refetchQueries({ queryKey: listStagesQueryKey, exact: true });
      }
    } catch {
      return;
    }
  }

  async watchWarehouses(opts?: {
    refreshHook?: () => void;
    onWarehouseEvent?: (warehouse: Warehouse) => void;
  }) {
    const warehousesInput = create(ListWarehousesRequestSchema, { project: this.project });
    const listWarehousesQueryKey = createConnectQueryKey({
      schema: listWarehouses,
      input: warehousesInput,
      cardinality: 'finite',
      transport: transportWithAuth
    });
    const refresh = {} as { [key: string]: boolean };

    try {
      while (!this.cancel.signal.aborted) {
        const warehousesResponse = this.client.getQueryData(listWarehousesQueryKey) as
          | ListWarehousesResponse
          | undefined;
        const stream = this.promiseClient.watchWarehouses(
          {
            project: this.project,
            resourceVersion: warehousesResponse?.resourceVersion || ''
          },
          { signal: this.cancel.signal }
        );

        const result = await ProcessEvents(
          stream,
          () => {
            const data = this.client.getQueryData(listWarehousesQueryKey);

            return (data as ListWarehousesResponse)?.warehouses || [];
          },
          (e) => e.warehouse as Warehouse,
          (warehouse, data) => {
            // refetch freight if necessary
            const refreshRequest = warehouse?.metadata?.annotations['kargo.akuity.io/refresh'];
            const refreshStatus = warehouse?.status?.lastHandledRefresh;
            const refreshing = refreshRequest !== undefined && refreshRequest !== refreshStatus;
            if (refreshing) {
              refresh[warehouse?.metadata?.name || ''] = true;
            } else if (refresh[warehouse?.metadata?.name || '']) {
              delete refresh[warehouse?.metadata?.name || ''];
              opts?.refreshHook?.();
            }

            // update Warehouse list
            this.client.setQueryData(listWarehousesQueryKey, {
              warehouses: data,
              resourceVersion:
                warehouse.metadata?.resourceVersion ||
                (this.client.getQueryData(listWarehousesQueryKey) as ListWarehousesResponse)
                  ?.resourceVersion ||
                '',
              $typeName: 'akuity.io.kargo.service.v1alpha1.ListWarehousesResponse'
            });

            // update Warehouse details
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
                value: warehouse,
                case: 'warehouse'
              }
            });

            opts?.onWarehouseEvent?.(warehouse);
          }
        );
        if (result !== 'resourceVersionExpired') {
          return;
        }
        await this.client.refetchQueries({ queryKey: listWarehousesQueryKey, exact: true });
      }
    } catch {
      return;
    }
  }
}
