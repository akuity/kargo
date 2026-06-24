import { useMemo } from 'react';

import { useListPromotions } from '@ui/gen/api/v2/core/core';
import { Promotion } from '@ui/gen/api/v2/models';

export const usePromotionsByFreightCollection = (payload: { project: string; stage: string }) => {
  const promotionsQuery = useListPromotions(payload.project, { stage: payload.stage });

  return useMemo(() => {
    const promotions = promotionsQuery.data?.data?.items || [];

    const promotionsByFreightCollection: Record<string, Promotion> = {};

    for (const promotion of promotions) {
      const freightCollectionId = promotion?.status?.freightCollection?.id || '';

      if (freightCollectionId) {
        promotionsByFreightCollection[freightCollectionId] = promotion;
      }
    }

    return promotionsByFreightCollection;
  }, [promotionsQuery.data]);
};
