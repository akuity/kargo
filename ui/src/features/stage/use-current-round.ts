import { useMemo } from 'react';

import { isStageTargetAware } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';

import { useLatestRollouts, useStageLatestRollouts } from './use-promotion-rollouts';
import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import { currentRound, currentRounds } from './utils/promotion-request';
import { Round, rolloutRound, rolloutRounds, stageUsesRollouts } from './utils/promotion-rollout';

// useCurrentRound resolves the round that best describes what a target-aware
// Stage is doing right now. It returns nothing for a classic Stage, which has
// no rounds.
//
// The round is a PromotionRollout for a Stage that fans out through rollouts
// and a PromotionRequest for any other; see stageUsesRollouts. The lists it
// reads are the ones the Promotions tab and the pipeline read, under the same
// query keys, so they share their fetches.
export const useCurrentRound = (project: string, stage?: Stage): Round | undefined => {
  const stageName = stage?.metadata?.name || '';
  const targetAware = isStageTargetAware(stage);

  const { latest } = useStageLatestRollouts(project, stage);
  const usesRollouts = stageUsesRollouts(stage, latest);

  // A Stage on rollouts has no use for its PromotionRequests.
  const readRequests = !!project && !!stageName && targetAware && !usesRollouts;
  const listQuery = useListPromotionRequests(
    project,
    { stage: stageName },
    { query: { enabled: readRequests } }
  );
  useWatchPromotionRequests(project, stageName, readRequests && !listQuery.isLoading);

  return useMemo(() => {
    if (!targetAware) {
      return undefined;
    }
    return usesRollouts
      ? rolloutRound(latest)
      : currentRound(stage, listQuery.data?.data?.items || []);
  }, [targetAware, usesRollouts, latest, stage, listQuery.data]);
};

// useCurrentRounds resolves the round every target-aware Stage is running now,
// keyed by Stage name. A surface summarizing a whole fleet needs all of them at
// once, so it reads the Project's PromotionRollouts and PromotionRequests in
// one query each rather than one per Stage.
export const useCurrentRounds = (
  project: string,
  stages: Stage[]
): Record<string, Round | undefined> => {
  const enabled = !!project && stages.length > 0;

  const { byStage } = useLatestRollouts(project, stages);

  const listQuery = useListPromotionRequests(project, undefined, { query: { enabled } });
  useWatchPromotionRequests(project, undefined, enabled && !listQuery.isLoading);

  return useMemo(() => {
    const fleet = stages.filter(isStageTargetAware);
    return {
      ...currentRounds(fleet, listQuery.data?.data?.items || []),
      ...rolloutRounds(fleet, byStage)
    };
  }, [stages, byStage, listQuery.data]);
};
