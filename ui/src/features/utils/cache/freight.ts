import { createConnectQueryKey } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import {
  getFreight,
  queryFreight
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

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
  }
};
