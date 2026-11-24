import { useMemo } from 'react';
import { useParams } from 'react-router-dom';

import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';

import { RoundsTable } from './rounds-table';
import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import { promotionRequestCompareFn } from './utils/promotion-request';
import { roundPath } from './utils/promotion-rollout';

type Props = {
  projectName: string;
  stageName: string;
};

/**
 * PromotionRequests is the Promotions tab of a target-aware Stage that fans
 * out through PromotionRequests: the requests the Stage has made, newest
 * first. They are Kubernetes resources, so the list is kept current by a
 * watch.
 */
export const PromotionRequests = ({ projectName, stageName }: Props) => {
  const { name: routeProjectName } = useParams();
  const project = routeProjectName || projectName;

  const listQuery = useListPromotionRequests(
    projectName,
    { stage: stageName },
    { query: { enabled: !!projectName && !!stageName } }
  );

  useWatchPromotionRequests(projectName, stageName, !listQuery.isLoading);

  const promotionRequests = useMemo(
    () => [...(listQuery.data?.data?.items || [])].sort(promotionRequestCompareFn),
    [listQuery.data]
  );

  return (
    <RoundsTable
      project={project}
      kind='request'
      rounds={promotionRequests}
      loading={listQuery.isLoading}
      pathOf={(round) => roundPath(project, stageName, round)}
    />
  );
};
