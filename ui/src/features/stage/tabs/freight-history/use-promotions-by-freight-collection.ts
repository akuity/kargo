import { useQuery } from '@connectrpc/connect-query';
import { useMemo } from 'react';

import { listPromotions } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

export const usePromotionsByFreightCollection = (payload: { project: string; stage: string }) => {
  const promotionsQuery = useQuery(listPromotions, payload);

  return useMemo(() => {
    const promotions = promotionsQuery.data?.promotions || [];

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
