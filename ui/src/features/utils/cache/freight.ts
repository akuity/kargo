import { createConnectQueryKey } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { transportWithAuth } from '@ui/config/transport';
import {
  getFreight,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { QueryFreightResponse } from '@ui/gen/api/service/v1alpha1/service_pb';

export default {
  refetch: () => {
    queryClient.refetchQueries({
      queryKey: createConnectQueryKey({
        schema: getFreight,
        cardinality: 'finite'
      })
    });
  },
  refetchQueryFreight: () => {
    queryClient.refetchQueries({
      exact: false,
      queryKey: createConnectQueryKey({
        schema: queryFreight,
        cardinality: 'finite'
      })
    });
  },
  get: (project: string) => {
    const data = queryClient.getQueryData(
      createConnectQueryKey({
        schema: queryFreight,
        input: {
          project
        },
        cardinality: 'finite',
        transport: transportWithAuth
      })
    ) as QueryFreightResponse;

    return data;
  }
};
