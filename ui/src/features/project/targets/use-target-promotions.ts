import { useInfiniteQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

import { getListTargetPromotionsQueryKey, listTargetPromotions } from '@ui/gen/api/v2/core/core';
import { TargetPromotion } from '@ui/gen/api/v2/models';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { orEmpty, polled } from './database-reads';

const inFlightPhases = ['Pending', 'Running'];

// useTargetPromotions reads one Target's Promotions, newest first, a page at
// a time: its history, from every Stage that has promoted to it. The pages
// read so far are polled while any of them is in flight.
export const useTargetPromotions = (project: string, target: string) => {
  const enabled = !!project && !!target;
  const query = useInfiniteQuery({
    queryKey: [...getListTargetPromotionsQueryKey(project, target), 'pages'],
    queryFn: ({ pageParam }) =>
      orEmpty(
        async () =>
          (
            await listTargetPromotions(
              project,
              target,
              pageParam ? { continue: pageParam } : undefined
            )
          ).data,
        { items: [] }
      ),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.continue || undefined,
    enabled,
    meta: polled,
    refetchInterval: (current) =>
      (current.state.data?.pages || []).some((page) =>
        (page.items || []).some((promotion) =>
          inFlightPhases.includes(promotion.phase || 'Pending')
        )
      )
        ? TARGETS_POLL_INTERVAL_MS
        : false
  });

  // A Promotion created between the reads of two pages moves every older one
  // down a place, so the same Promotion can close one page and open the next.
  const promotions = useMemo(() => {
    const byName = new Map<string, TargetPromotion>();
    for (const page of query.data?.pages || []) {
      for (const promotion of page.items || []) {
        byName.set(promotion.name || '', promotion);
      }
    }
    return [...byName.values()];
  }, [query.data]);

  return {
    promotions,
    isLoading: enabled && query.isLoading,
    hasMore: !!query.hasNextPage,
    isLoadingMore: query.isFetchingNextPage,
    loadMore: query.fetchNextPage
  };
};
