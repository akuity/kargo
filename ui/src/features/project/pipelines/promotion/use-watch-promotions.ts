import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { getListPromotionsQueryKey, listPromotionsResponse } from '@ui/gen/api/v2/core/core';
import { Promotion } from '@ui/gen/api/v2/models';

import { readSSEStream, upsertOrDelete } from '../watch-utils';

export const useWatchPromotions = (project: string, stage: string) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !stage) {
      return;
    }

    const abort = new AbortController();
    const params = new URLSearchParams({ watch: 'true', stage });
    const url = `/v1beta1/projects/${encodeURIComponent(project)}/promotions?${params}`;
    const listKey = getListPromotionsQueryKey(project, { stage });

    (async () => {
      for await (const event of readSSEStream<Promotion>(url, abort.signal)) {
        const promotion = event.object;

        client.setQueryData(listKey, (old: listPromotionsResponse | undefined) => {
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
    })();

    return () => abort.abort();
  }, [project, stage, client]);
};
