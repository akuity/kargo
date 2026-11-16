import { useMemo } from 'react';

import { useGetPromotionRollout } from '@ui/gen/api/v2/core/core';

import { RoundDetails } from '../promotion-request/round-details';
import { ROLLOUTS_POLL_INTERVAL_MS } from '../use-promotion-rollouts';
import { isRolloutFinished, rolloutToRound } from '../utils/promotion-rollout';

/**
 * PromotionRolloutDetails is one PromotionRollout, read by its number among
 * its Stage's rollouts and shown as the round of fan-out it is.
 *
 * A rollout lives in the database, which offers no watch, so it is read again
 * on a timer until it finishes -- and only for as long as reading it works: a
 * failing read left on the timer would raise the global error notification on
 * every tick.
 */
export const PromotionRolloutDetails = ({
  project,
  stageName,
  number
}: {
  project: string;
  // The Stage whose tab this is rendered in, and whose rollouts the number
  // counts among.
  stageName: string;
  number: number;
}) => {
  // A number the route could not have meant is not asked for at all.
  const valid = Number.isInteger(number) && number > 0;

  const rolloutQuery = useGetPromotionRollout(project, stageName, number, {
    query: {
      enabled: valid && !!project && !!stageName,
      // A rollout that does not exist is reported by the view itself, so it
      // needs no notification on top.
      meta: { silent404: true },
      refetchInterval: (query) =>
        query.state.status === 'error' || isRolloutFinished(query.state.data?.data)
          ? false
          : ROLLOUTS_POLL_INTERVAL_MS
    }
  });

  const rollout = rolloutQuery.data?.data;
  const round = useMemo(() => rollout && rolloutToRound(rollout), [rollout]);
  const name = rollout?.metadata?.name;

  return (
    <RoundDetails
      project={project}
      stageName={stageName}
      kind='rollout'
      identifier={name ? `#${number} · ${name}` : `#${number}`}
      round={round}
      isLoading={rolloutQuery.isLoading}
      error={rolloutQuery.error}
    />
  );
};
