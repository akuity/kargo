import { Button, Flex } from 'antd';
import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v2/models';

import { RoundsTable } from './rounds-table';
import { useStageRollouts } from './use-promotion-rollouts';
import { rolloutToRound, roundPath } from './utils/promotion-rollout';

/**
 * PromotionRollouts is the Promotions tab of a target-aware Stage that fans
 * out through PromotionRollouts: the rollouts the Stage has run, newest first.
 *
 * Rollouts live in the database rather than in Kubernetes, so there is nothing
 * to watch: the list is read a page at a time, and read again on a timer while
 * the Stage has a rollout in flight.
 */
export const PromotionRollouts = ({ project, stage }: { project: string; stage: Stage }) => {
  const stageName = stage.metadata?.name || '';
  const { rollouts, isLoading, hasMore, isLoadingMore, loadMore } = useStageRollouts(
    project,
    stage
  );

  const rounds = useMemo(() => rollouts.map(rolloutToRound), [rollouts]);

  return (
    <RoundsTable
      project={project}
      kind='rollout'
      rounds={rounds}
      loading={isLoading}
      pathOf={(round) => roundPath(project, stageName, round)}
      footer={
        hasMore && (
          <Flex justify='center'>
            <Button size='small' loading={isLoadingMore} onClick={() => loadMore()}>
              Load older rollouts
            </Button>
          </Flex>
        )
      }
    />
  );
};
