import { useMemo } from 'react';

import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useListPromotionRollouts } from '@ui/gen/api/v2/core/core';
import { PromotionRollout, Stage } from '@ui/gen/api/v2/models';

import { PROMOTION_ROLLOUTS_POLL_INTERVAL_MS } from './constants';
import { currentRound, currentRounds } from './utils/promotion-rollout';

// useCurrentRound resolves the PromotionRollout that best describes what a
// target-aware Stage is doing right now. It returns nothing for a classic
// Stage, which has no rounds.
//
// The list it reads is the same one the Promotions tab shows, under the same
// query key, so the two share one fetch and one poll.
export const useCurrentRound = (project: string, stage?: Stage): PromotionRollout | undefined => {
  const stageName = stage?.metadata?.name || '';
  const targetAware = isStageTargetAware(stage);

  const listQuery = useListPromotionRollouts(
    project,
    { stage: stageName },
    {
      query: {
        enabled: !!project && !!stageName && targetAware,
        refetchInterval: PROMOTION_ROLLOUTS_POLL_INTERVAL_MS
      }
    }
  );

  return useMemo(
    () => (targetAware ? currentRound(stage, listQuery.data?.data?.items || []) : undefined),
    [targetAware, stage, listQuery.data]
  );
};

// useCurrentRounds resolves the round every target-aware Stage is running now,
// keyed by Stage name, from one project-wide list of PromotionRollouts. A
// surface summarizing a whole fleet needs all of them at once, so it reads the
// Project's rollouts in a single query rather than one per Stage.
export const useCurrentRounds = (
  project: string,
  stages: Stage[]
): Record<string, PromotionRollout | undefined> => {
  const enabled = !!project && stages.length > 0;

  const listQuery = useListPromotionRollouts(project, undefined, {
    query: { enabled, refetchInterval: PROMOTION_ROLLOUTS_POLL_INTERVAL_MS }
  });

  return useMemo(
    () => currentRounds(stages.filter(isStageTargetAware), listQuery.data?.data?.items || []),
    [stages, listQuery.data]
  );
};
