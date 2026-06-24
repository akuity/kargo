import { queryClient } from '@ui/config/query-client';
import { getQueryFreightsRestQueryKey, queryFreightsRestResponse } from '@ui/gen/api/v2/core/core';

export default {
  refetch: (project: string) => {
    // Refetch the freight detail views currently on screen for this project.
    // Their GET key is `[`/v1beta1/projects/${project}/freight/${name}`]`, so a
    // prefix predicate matches them while excluding the freight LIST query
    // (`.../freight`, no trailing slash), which refetchQueryFreight handles. A
    // plain `queryKey` filter can't express this: getGetFreightQueryKey(project)
    // yields the literal `.../freight/undefined`, which matches nothing.
    //
    // type: 'active' is required. useWatchFreight seeds individual GET caches
    // via setQueryData (use-watch-freight.ts), leaving orphaned entries with no
    // queryFn for any freight whose detail view is closed; refetching those
    // errors ("Missing queryFn"). Limiting to active (observer-backed) queries
    // refetches only mounted views -- which also matches main's effective
    // behavior, where getFreight entries existed only while observed.
    const prefix = `/v1beta1/projects/${project}/freight/`;
    queryClient.refetchQueries({
      type: 'active',
      predicate: (query) => {
        const key = query.queryKey[0];
        return typeof key === 'string' && key.startsWith(prefix);
      }
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
