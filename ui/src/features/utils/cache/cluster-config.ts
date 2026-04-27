import { queryClient } from '@ui/config/query-client';
import { getGetClusterConfigQueryKey } from '@ui/gen/api/v2/system/system';

export default {
  refetch: () =>
    queryClient.refetchQueries({
      queryKey: getGetClusterConfigQueryKey()
    })
};
