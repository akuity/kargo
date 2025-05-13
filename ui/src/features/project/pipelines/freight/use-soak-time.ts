import { Duration } from 'date-fns';
import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { durationToSeconds } from './duration-to-seconds';
import { getSoakTime } from './get-soak-time';

export const useSoakTime = (freights: Freight[]) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  return useMemo(() => {
    const soakTimes: Record<string, Duration> = {};

    if (
      actionContext?.action?.type !== IAction.PROMOTE &&
      actionContext?.action?.type !== IAction.PROMOTE_DOWNSTREAM
    ) {
      return soakTimes;
    }

    const currentlyPromotingStage = actionContext?.action?.stage;
    const currentlyPromotingStageName = actionContext?.action?.stage?.metadata?.name || '';
    const sourcesOfCurrentlyPromotingStage: string[] = [];

    for (const [source, dest] of Object.entries(dictionaryContext?.subscribersByStage || {})) {
      if (dest.has(currentlyPromotingStageName)) {
        sourcesOfCurrentlyPromotingStage.push(source);
      }
    }

    for (const freight of freights) {
      const freightOrigin = freight?.origin?.name || '';
      const freightInStages = Object.keys(freight?.status?.currentlyIn || {});

      const requireSoakTime = currentlyPromotingStage?.spec?.requestedFreight?.find(
        (f) => f?.origin?.name === freightOrigin
      )?.sources?.requiredSoakTime;

      const soakTimesForFreight: Duration[] = [];
      for (const stage of freightInStages) {
        if (!sourcesOfCurrentlyPromotingStage.includes(stage)) {
          continue;
        }

        const stageDetails = dictionaryContext?.stageByName?.[stage];

        if (stageDetails && requireSoakTime) {
          const soakTime = getSoakTime({
            freight,
            freightInStage: dictionaryContext?.stageByName?.[stage],
            requiredSoakTime: requireSoakTime
          });

          if (soakTime) {
            soakTimesForFreight.push(soakTime);
          }
        }
      }

      if (soakTimesForFreight.length > 0) {
        const maxSoakTime = soakTimesForFreight.reduce((max, curr) =>
          durationToSeconds(curr) > durationToSeconds(max) ? curr : max
        );

        soakTimes[freight?.metadata?.name || ''] = maxSoakTime;
      }
    }

    return soakTimes;
  }, [actionContext, freights, dictionaryContext]);
};
