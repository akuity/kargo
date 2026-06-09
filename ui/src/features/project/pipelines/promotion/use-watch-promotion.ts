import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { getGetPromotionQueryKey, getPromotionResponse } from '@ui/gen/api/v2/core/core';
import { Promotion } from '@ui/gen/api/v2/models';

import { readSSEStream } from '../watch-utils';

export const useWatchPromotion = (project: string, promotion: string) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !promotion) {
      return;
    }

    const abort = new AbortController();
    const url = `/v1beta1/projects/${encodeURIComponent(project)}/promotions/${encodeURIComponent(promotion)}?watch=true`;
    const promotionKey = getGetPromotionQueryKey(project, promotion);

    (async () => {
      for await (const event of readSSEStream<Promotion>(url, abort.signal)) {
        const p = event.object;

        if (event.type === 'DELETED') {
          client.removeQueries({ queryKey: promotionKey });
        } else {
          client.setQueryData(promotionKey, (old: getPromotionResponse | undefined) =>
            old ? { ...old, data: p } : old
          );
        }
      }
    })();

    return () => abort.abort();
  }, [project, promotion]);
};
