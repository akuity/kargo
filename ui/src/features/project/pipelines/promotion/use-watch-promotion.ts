import { createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { transportWithAuth } from '@ui/config/transport';
import { getPromotion } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { KargoService, RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';

export const useWatchPromotion = (project: string, promotion: string) => {
  const client = useQueryClient();

  useEffect(() => {
    const cancel = new AbortController();

    const watchPromotion = async () => {
      const promiseClient = createClient(KargoService, transportWithAuth);
      const stream = promiseClient.watchPromotion(
        {
          project,
          name: promotion
        },
        { signal: cancel.signal }
      );

      for await (const e of stream) {
        const updatedPromotion = e.promotion;

        if (promotion && updatedPromotion) {
          const promotionQueryKey = createConnectQueryKey({
            cardinality: 'finite',
            schema: getPromotion,
            input: {
              project,
              name: promotion
            },
            transport: transportWithAuth
          });

          client.setQueryData(promotionQueryKey, {
            result: {
              value: updatedPromotion,
              case: 'promotion'
            },
            $typeName: 'akuity.io.kargo.service.v1alpha1.GetPromotionResponse'
          });

          client.refetchQueries({
            queryKey: createConnectQueryKey({
              cardinality: 'finite',
              schema: getPromotion,
              input: {
                project,
                name: promotion,
                format: RawFormat.YAML
              },
              transport: transportWithAuth
            })
          });
        }
      }
    };

    watchPromotion();

    return () => cancel.abort();
  }, [project, promotion]);
};
