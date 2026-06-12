import { queryClient } from '@ui/config/query-client';
import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';

export default {
  refetch: (project: string) =>
    queryClient.refetchQueries({
      queryKey: getGetProjectConfigQueryKey(project)
    })
};
