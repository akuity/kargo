import { useMemo } from 'react';

import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';

import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import { currentRound, currentRounds } from './utils/promotion-request';

// useCurrentRound resolves the PromotionRequest that best describes what a
// target-aware Stage is doing right now. It returns nothing for a classic
// Stage, which has no rounds.
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

  return useMemo(
    () => (targetAware ? currentRound(stage, listQuery.data?.data?.items || []) : undefined),
    [targetAware, stage, listQuery.data]
  );
};

// useCurrentRounds resolves the round every target-aware Stage is running now,
// keyed by Stage name, from one project-wide list of PromotionRequests. A
// surface summarizing a whole fleet needs all of them at once, so it reads the
// Project's requests in a single query rather than one per Stage.
export const useCurrentRounds = (
  project: string,
  stages: Stage[]
): Record<string, PromotionRequest | undefined> => {
  const enabled = !!project && stages.length > 0;

  const listQuery = useListPromotionRequests(project, undefined, { query: { enabled } });
  useWatchPromotionRequests(project, undefined, enabled && !listQuery.isLoading);

  return useMemo(
    () => currentRounds(stages.filter(isStageTargetAware), listQuery.data?.data?.items || []),
    [stages, listQuery.data]
  );
};
