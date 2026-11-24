import { useGetPromotionRequest } from '@ui/gen/api/v2/core/core';

import { isPromotionRequestPhaseTerminal } from '../utils/promotion-request';

import { RoundDetails } from './round-details';

// A round that is still fanning out keeps changing. Its children are watched
// and so stay current on their own, but the request itself has no single-object
// watch, so it is re-read on a timer for as long as it can still change -- and
// only for as long as reading it works. A failing read left on the timer would
// raise the global error notification every tick, for as long as the tab stays
// open.
const refetchIntervalMs = 5000;

/**
 * PromotionRequestDetails is one PromotionRequest, read by name and shown as
 * the round of fan-out it is.
 */
export const PromotionRequestDetails = ({
  project,
  stageName,
  promotionRequestName
}: {
  project: string;
  // The Stage whose tab this is rendered in. It comes from the route rather
  // than from the request, so the way back is there before the request loads.
  stageName: string;
  promotionRequestName: string;
}) => {
  const requestQuery = useGetPromotionRequest(project, promotionRequestName, {
    query: {
      // A round that no longer exists is reported by the view itself, so it
      // needs no notification on top.
      meta: { silent404: true },
      refetchInterval: (query) =>
        query.state.status === 'error' ||
        isPromotionRequestPhaseTerminal(query.state.data?.data?.status?.phase)
          ? false
          : refetchIntervalMs
    }
  });

  return (
    <RoundDetails
      project={project}
      stageName={stageName}
      kind='request'
      identifier={promotionRequestName}
      round={requestQuery.data?.data}
      isLoading={requestQuery.isLoading}
      error={requestQuery.error}
    />
  );
};
