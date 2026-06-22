import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { getListPromotionsQueryKey, listPromotionsResponse } from '@ui/gen/api/v2/core/core';
import { Promotion } from '@ui/gen/api/v2/models';

import { runSeededWatch, upsertOrDelete } from '../watch-utils';

// enabled gates the watch on the initial list having loaded, so it never opens
// before a seed resourceVersion is available — otherwise it would start an
// unseeded watch that replays every Promotion. (Unlike the pipeline page, this
// page has no loading gate above the watch, so the caller must pass it.)
export const useWatchPromotions = (project: string, stage: string, enabled = true) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !stage || !enabled) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListPromotionsQueryKey(project, { stage });

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listPromotionsResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true', stage });
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/promotions?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, promotion: Promotion) => {
      client.setQueryData(listKey, (old: listPromotionsResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: {
            ...old.data,
            items: upsertOrDelete(old.data.items ?? [], promotion, type)
          }
        };
      });
    };

    runSeededWatch<Promotion>({
      signal: abort.signal,
      buildUrl,
      seedResourceVersion,
      relist,
      onEvent
    });

    return () => abort.abort();
  }, [project, stage, client, enabled]);
};
