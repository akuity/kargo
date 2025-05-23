import { createClient } from '@connectrpc/connect';
import { createConnectQueryKey } from '@connectrpc/connect-query';
import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { transportWithAuth } from '@ui/config/transport';
import { queryCache } from '@ui/features/utils/cache';
import { queryFreight } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { KargoService } from '@ui/gen/api/service/v1alpha1/service_pb';

export const useWatchFreight = (project: string) => {
  const client = useQueryClient();

  useEffect(() => {
    const cancel = new AbortController();

    const watchFreight = async () => {
      const promiseClient = createClient(KargoService, transportWithAuth);

      const stream = promiseClient.watchFreight(
        {
          project
        },
        { signal: cancel.signal }
      );

      for await (const e of stream) {
        const freight = e.freight;

        if (!freight) {
          continue;
        }

        let currentFreight = queryCache.freight.get(project);

        if (e.type !== 'DELETED') {
          const exist = currentFreight?.groups?.['']?.freight?.find(
            (f) => f?.metadata?.name === freight?.metadata?.name
          );

          if (!exist) {
            currentFreight?.groups?.['']?.freight?.push(freight);
          } else {
            currentFreight = {
              ...currentFreight,
              groups: {
                ...currentFreight?.groups,
                '': {
                  ...currentFreight?.groups?.[''],
                  freight: currentFreight?.groups?.['']?.freight?.map((f) => {
                    if (f?.metadata?.name === freight?.metadata?.name) {
                      return freight;
                    }

                    return f;
                  })
                }
              }
            };
          }

          const queryFreightKey = createConnectQueryKey({
            cardinality: 'finite',
            schema: queryFreight,
            input: {
              project
            },
            transport: transportWithAuth
          });

          client.setQueryData(queryFreightKey, currentFreight);
        }
      }
    };

    watchFreight();

    return () => cancel.abort();
  }, [project]);
};
