import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getListPromotionRequestsQueryKey,
  listPromotionRequestsResponse
} from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';

import { runSeededWatch, upsertOrDelete } from '../project/pipelines/watch-utils';

export const useWatchPromotionRequests = (project: string, stage: string, enabled = true) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !stage || !enabled) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListPromotionRequestsQueryKey(project, { stage });

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listPromotionRequestsResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true', stage });
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/promotion-requests?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, promotionRequest: PromotionRequest) => {
      client.setQueryData(listKey, (old: listPromotionRequestsResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: {
            ...old.data,
            items: upsertOrDelete(old.data.items ?? [], promotionRequest, type)
          }
        };
      });
    };

    runSeededWatch<PromotionRequest>({
      signal: abort.signal,
      buildUrl,
      seedResourceVersion,
      relist,
      onEvent
    });

    return () => abort.abort();
  }, [project, stage, client, enabled]);
};
