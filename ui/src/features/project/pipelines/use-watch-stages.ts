import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetStageQueryKey,
  getListStagesQueryKey,
  getStageResponse,
  listStagesResponse
} from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

import { readSSEStream, upsertOrDelete } from './watch-utils';

export const useWatchStages = (
  project: string,
  onStageEvent?: (stage: Stage) => void,
  warehouses?: string[]
) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const abort = new AbortController();

    const params = new URLSearchParams({ watch: 'true' });
    for (const wh of warehouses || []) {
      params.append('freightOrigins', wh);
    }
    const url = `/v1beta1/projects/${encodeURIComponent(project)}/stages?${params}`;
    const listKey = getListStagesQueryKey(project, { freightOrigins: warehouses || [] });

    (async () => {
      for await (const event of readSSEStream<Stage>(url, abort.signal)) {
        const stage = event.object;

        client.setQueryData(listKey, (old: listStagesResponse | undefined) => {
          if (!old?.data) {
            return old;
          }
          return {
            ...old,
            data: { ...old.data, items: upsertOrDelete(old.data.items ?? [], stage, event.type) }
          };
        });

        const stageKey = getGetStageQueryKey(project, stage.metadata?.name);
        if (event.type === 'DELETED') {
          client.removeQueries({ queryKey: stageKey });
        } else {
          client.setQueryData(stageKey, (old: getStageResponse | undefined) =>
            old ? { ...old, data: stage } : old
          );
          onStageEvent?.(stage);
        }
      }
    })();

    return () => abort.abort();
  }, [project, (warehouses || []).join(',')]);
};
