import { queryClient } from '@ui/config/query-client';
import {
  getGetFreightQueryKey,
  getQueryFreightsRestQueryKey,
  queryFreightsRestResponse
} from '@ui/gen/api/v2/core/core';

export default {
  refetch: (project: string) => {
    queryClient.refetchQueries({
      exact: false,
      queryKey: getGetFreightQueryKey(project)
    });
  },
  refetchQueryFreight: (project: string) => {
    queryClient.refetchQueries({
      exact: false,
      queryKey: getQueryFreightsRestQueryKey(project)
    });
  },
  get: (project: string) => {
    return queryClient.getQueryData(
      getQueryFreightsRestQueryKey(project)
    ) as queryFreightsRestResponse;
  }
};
