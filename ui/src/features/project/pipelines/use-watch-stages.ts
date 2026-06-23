import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getGetStageQueryKey,
  getListStagesQueryKey,
  getStageResponse,
  listStagesResponse
} from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

import { batchEmitter, runSeededWatch, upsertOrDelete } from './watch-utils';

export const useWatchStages = (
  project: string,
  onStagesEvent?: (stages: Stage[]) => void,
  warehouses?: string[]
) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListStagesQueryKey(project, { freightOrigins: warehouses || [] });

    // Batch bursts of stage events so the graph applies them all at once rather
    // than once per event. Keying by name keeps every distinct stage's latest
    // update — a plain debounce would drop all but the last object in a burst.
    const emitStageEvents = batchEmitter(
      (stages: Stage[]) => onStagesEvent?.(stages),
      (stage) => stage.metadata?.name ?? ''
    );

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listStagesResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true' });
      for (const wh of warehouses || []) {
        params.append('freightOrigins', wh);
      }
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/stages?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, stage: Stage) => {
      client.setQueriesData(
        { exact: false, queryKey: listKey },
        (old: listStagesResponse | undefined) => {
          if (!old?.data) {
            return old;
          }
          return {
            ...old,
            data: { ...old.data, items: upsertOrDelete(old.data.items ?? [], stage, type) }
          };
        }
      );

      const stageKey = getGetStageQueryKey(project, stage.metadata?.name);
      if (type === 'DELETED') {
        client.removeQueries({ queryKey: stageKey });
      } else {
        client.setQueriesData(
          { exact: false, queryKey: stageKey },
          (old: getStageResponse | undefined) =>
            old
              ? {
                  ...old,
                  data: {
                    // WATCH ENDPOINT STAGE COMES WITHOUT kind, apiVersion etc..
                    // SO WE NEED TO PRESERVE IT FROM INITIAL DATA
                    ...old?.data,
                    ...stage
                  }
                }
              : old
        );

        emitStageEvents.call(stage);
      }
    };

    runSeededWatch<Stage>({ signal: abort.signal, buildUrl, seedResourceVersion, relist, onEvent });

    return () => {
      abort.abort();
      emitStageEvents.cancel();
    };
  }, [project, (warehouses || []).join(','), client]);
};
