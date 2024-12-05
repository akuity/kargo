import { create } from '@bufbuild/protobuf';
import { Client, createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { QueryClient } from '@tanstack/react-query';

import { transportWithAuth } from '@ui/config/transport';
import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import {
  getStage,
  listStages,
  listWarehouses
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import {
  GetStageRequestSchema,
  KargoService,
  ListStagesRequestSchema,
  ListStagesResponse,
  ListWarehousesRequestSchema,
  ListWarehousesResponse
} from '@ui/gen/service/v1alpha1/service_pb';
import { Stage, Warehouse } from '@ui/gen/v1alpha1/generated_pb';

async function ProcessEvents<T extends { type: string }, S extends { metadata?: ObjectMeta }>(
  stream: AsyncIterable<T>,
  getData: () => S[],
  getter: (e: T) => S,
  callback: (item: S, data: S[]) => void
) {
  for await (const e of stream) {
    let data = getData();
    const index = data.findIndex((item) => item.metadata?.name === getter(e).metadata?.name);
    if (e.type === 'DELETED') {
      if (index !== -1) {
        data = [...data.slice(0, index), ...data.slice(index + 1)];
      }
    } else {
      if (index === -1) {
        data = [...data, getter(e)];
      } else {
        data = [...data.slice(0, index), getter(e), ...data.slice(index + 1)];
      }
    }

    callback(getter(e), data);
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
      (stage, data) => {
        // update Stages list
        const listStagesQueryKey = createConnectQueryKey({
          schema: listStages,
          input: create(ListStagesRequestSchema, { project: this.project }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(listStagesQueryKey, { stages: data });

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
        this.client.setQueryData(getStageQueryKey, { stage });

        onStageEvent?.(stage);
      }
    );
  }

  async watchWarehouses(refreshHook: () => void) {
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
          refreshHook();
        }

        // update Warehouse list
        const listWarehousesQueryKey = createConnectQueryKey({
          schema: listWarehouses,
          input: create(ListWarehousesRequestSchema, {
            project: this.project
          }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(listWarehousesQueryKey, { warehouses: data });

        // update Warehouse details
        const getWarehouseQueryKey = createConnectQueryKey({
          schema: getStage,
          input: create(GetStageRequestSchema, {
            project: this.project,
            name: warehouse.metadata?.name
          }),
          cardinality: 'finite',
          transport: transportWithAuth
        });
        this.client.setQueryData(getWarehouseQueryKey, { warehouse });
      }
    );
  }
}
