import { createConnectQueryKey } from '@connectrpc/connect-query';

import { queryClient } from '@ui/config/query-client';
import { getClusterConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export default {
  refetch: () =>
    queryClient.refetchQueries({
      queryKey: createConnectQueryKey({
        schema: getClusterConfig,
        cardinality: 'finite'
      })
    })
};
