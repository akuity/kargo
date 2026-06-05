import { QueryClient } from '@tanstack/react-query';

import { authTokenKey } from '@ui/config/auth';
import {
  getFreightResponse,
  getGetFreightQueryKey,
  getGetPromotionQueryKey,
  getGetStageQueryKey,
  getGetWarehouseQueryKey,
  getListPromotionsQueryKey,
  getListStagesQueryKey,
  getListWarehousesQueryKey,
  getPromotionResponse,
  getQueryFreightsRestQueryKey,
  getStageResponse,
  getWarehouseResponse,
  listPromotionsResponse,
  listStagesResponse,
  listWarehousesResponse,
  queryFreightsRestResponse
} from '@ui/gen/api/v2/core/core';
import { Freight, Promotion, Stage, Warehouse } from '@ui/gen/api/v2/models';

const getBaseUrl = () => (import.meta.env.VITE_API_URL as string | undefined) || '';

type SSEWatchEvent<T> = { type: string; object: T };

async function* readSSEStream<T>(
  url: string,
  signal: AbortSignal
): AsyncGenerator<SSEWatchEvent<T>> {
  const token = localStorage.getItem(authTokenKey);
  const response = await fetch(`${getBaseUrl()}${url}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    signal
  });

  if (!response.ok || !response.body) {
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop() ?? '';

      for (const part of parts) {
        const dataLine = part.split('\n').find((l) => l.startsWith('data: '));
        if (!dataLine) {
          continue;
        }
        try {
          yield JSON.parse(dataLine.slice(6)) as SSEWatchEvent<T>;
        } catch (_) {
          // skip malformed events
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

function upsertOrDelete<T extends { metadata?: { name?: string } }>(
  items: T[],
  item: T,
  eventType: string
): T[] {
  const index = items.findIndex((i) => i.metadata?.name === item.metadata?.name);
  if (eventType === 'DELETED') {
    return index !== -1 ? [...items.slice(0, index), ...items.slice(index + 1)] : items;
  }
  // ADDED or MODIFIED
  return index !== -1
    ? [...items.slice(0, index), item, ...items.slice(index + 1)]
    : [...items, item];
}

export class Watcher {
  private cancel: AbortController;
  private _client: QueryClient;
  project: string;

  constructor(project: string, client: QueryClient) {
    this.cancel = new AbortController();
    this._client = client;
    this.project = project;
  }

  cancelWatch() {
    this.cancel.abort();
  }

  async watchStages(onStageEvent?: (stage: Stage) => void, warehouses?: string[]) {
    const params = new URLSearchParams({ watch: 'true' });
    for (const wh of warehouses || []) {
      params.append('freightOrigins', wh);
    }
    const url = `/v1beta1/projects/${encodeURIComponent(this.project)}/stages?${params}`;
    const listKey = getListStagesQueryKey(
      this.project,
      warehouses?.length ? { freightOrigins: warehouses } : {}
    );

    for await (const event of readSSEStream<Stage>(url, this.cancel.signal)) {
      const stage = event.object;

      // update list cache
      this._client.setQueryData(listKey, (old: listStagesResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: { ...old.data, items: upsertOrDelete(old.data.items ?? [], stage, event.type) }
        };
      });

      // update individual stage cache
      const stageKey = getGetStageQueryKey(this.project, stage.metadata?.name);
      if (event.type === 'DELETED') {
        this._client.removeQueries({ queryKey: stageKey });
      } else {
        this._client.setQueryData(stageKey, (old: getStageResponse | undefined) =>
          old ? { ...old, data: stage } : old
        );
        onStageEvent?.(stage);
      }
    }
  }

  async watchWarehouses(opts?: {
    refreshHook?: () => void;
    onWarehouseEvent?: (warehouse: Warehouse) => void;
  }) {
    const url = `/v1beta1/projects/${encodeURIComponent(this.project)}/warehouses?watch=true`;
    const listKey = getListWarehousesQueryKey(this.project);
    const pendingRefresh: Record<string, boolean> = {};

    for await (const event of readSSEStream<Warehouse>(url, this.cancel.signal)) {
      const warehouse = event.object;
      const name = warehouse.metadata?.name || '';

      // update list cache
      this._client.setQueryData(listKey, (old: listWarehousesResponse | undefined) => {
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
      });

      // update individual warehouse cache
      const warehouseKey = getGetWarehouseQueryKey(this.project, name);
      if (event.type === 'DELETED') {
        this._client.removeQueries({ queryKey: warehouseKey });
      } else {
        this._client.setQueryData(warehouseKey, (old: getWarehouseResponse | undefined) =>
          old ? { ...old, data: warehouse } : old
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
  }

  async watchFreights(params?: { origins?: string[]; stage?: string }) {
    const urlParams = new URLSearchParams({ watch: 'true' });
    for (const o of params?.origins || []) {
      urlParams.append('origins', o);
    }
    if (params?.stage) {
      urlParams.set('stage', params.stage);
    }
    const url = `/v1beta1/projects/${encodeURIComponent(this.project)}/freight?${urlParams}`;
    const listKey = getQueryFreightsRestQueryKey(this.project, params);

    for await (const event of readSSEStream<Freight>(url, this.cancel.signal)) {
      const freight = event.object;

      this._client.setQueryData(listKey, (old: queryFreightsRestResponse | undefined) => {
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

      const freightKey = getGetFreightQueryKey(this.project, freight.metadata?.name);
      if (event.type === 'DELETED') {
        this._client.removeQueries({ queryKey: freightKey });
      } else {
        this._client.setQueryData(freightKey, (old: getFreightResponse | undefined) =>
          old ? { ...old, data: freight } : old
        );
      }
    }
  }

  async watchPromotions(stage: string) {
    const params = new URLSearchParams({ watch: 'true' });
    if (stage) {
      params.set('stage', stage);
    }
    const url = `/v1beta1/projects/${encodeURIComponent(this.project)}/promotions?${params}`;
    const listKey = getListPromotionsQueryKey(this.project, { stage });

    for await (const event of readSSEStream<Promotion>(url, this.cancel.signal)) {
      const promotion = event.object;

      this._client.setQueryData(listKey, (old: listPromotionsResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: {
            ...old.data,
            items: upsertOrDelete(old.data.items ?? [], promotion, event.type)
          }
        };
      });
    }
  }

  async watchPromotion(name: string) {
    const url = `/v1beta1/projects/${encodeURIComponent(this.project)}/promotions/${encodeURIComponent(name)}?watch=true`;
    const promotionKey = getGetPromotionQueryKey(this.project, name);

    for await (const event of readSSEStream<Promotion>(url, this.cancel.signal)) {
      const promotion = event.object;

      if (event.type === 'DELETED') {
        this._client.removeQueries({ queryKey: promotionKey });
      } else {
        this._client.setQueryData(promotionKey, (old: getPromotionResponse | undefined) =>
          old ? { ...old, data: promotion } : old
        );
      }
    }
  }
}
