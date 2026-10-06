import { useMemo } from 'react';

import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';

import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import { promotionRequestCompareFn } from './utils/promotion-request';

// useCurrentRound resolves the PromotionRequest that best describes what a
// target-aware Stage is doing right now: the one its status names as current,
// else the one it names as last, else the newest it has at all -- the last two
// cover a Stage whose status has not caught up with its requests yet. It
// returns nothing for a classic Stage, which has no rounds.
//
// The list it reads is the same one the Promotions tab shows, under the same
// query key, so the two share one fetch and one watch.
export const useCurrentRound = (project: string, stage?: Stage): PromotionRequest | undefined => {
  const stageName = stage?.metadata?.name || '';
  const targetAware = isStageTargetAware(stage);

  const listQuery = useListPromotionRequests(
    project,
    { stage: stageName },
    { query: { enabled: !!project && !!stageName && targetAware } }
  );
  useWatchPromotionRequests(project, stageName, targetAware && !listQuery.isLoading);

  return useMemo(() => {
    if (!targetAware) {
      return undefined;
    }
    const requests = listQuery.data?.data?.items || [];
    const named = (name?: string) =>
      name ? requests.find((request) => request.metadata?.name === name) : undefined;
    return (
      named(stage?.status?.currentPromotionRequest?.name) ||
      named(stage?.status?.lastPromotionRequest?.name) ||
      [...requests].sort(promotionRequestCompareFn)[0]
    );
  }, [targetAware, listQuery.data, stage?.status]);
};
