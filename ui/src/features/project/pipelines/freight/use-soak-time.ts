import { Duration } from 'date-fns';
import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { durationToSeconds } from './duration-to-seconds';
import { getSoakTime } from './get-soak-time';

const soakTimeForPromotingStage = (payload: {
  stage: Stage;
  subscribersByStage: Record<string, Set<string>>;
  freights: Freight[];
  stageByName: Record<string, Stage>;
  soakTimesByFreight: Record<string, Duration>;
}) => {
  const currentlyPromotingStage = payload.stage;
  const currentlyPromotingStageName = currentlyPromotingStage?.metadata?.name || '';
  const sourcesOfCurrentlyPromotingStage: string[] = [];

  for (const [source, dest] of Object.entries(payload.subscribersByStage)) {
    if (dest.has(currentlyPromotingStageName)) {
      sourcesOfCurrentlyPromotingStage.push(source);
    }
  }

  for (const freight of payload.freights) {
    const freightName = freight?.metadata?.name || '';
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

      const stageDetails = payload.stageByName?.[stage];

      if (stageDetails && requireSoakTime) {
        const soakTime = getSoakTime({
          freight,
          freightInStage: payload.stageByName?.[stage],
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

      if (!payload.soakTimesByFreight[freightName]) {
        payload.soakTimesByFreight[freightName] = maxSoakTime;
      } else {
        const existingSoakTime = payload.soakTimesByFreight[freightName];
        payload.soakTimesByFreight[freightName] =
          durationToSeconds(existingSoakTime) > durationToSeconds(maxSoakTime)
            ? existingSoakTime
            : maxSoakTime;
      }
    }
  }

  return payload.soakTimesByFreight;
};

export const useSoakTime = (freights: Freight[]) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  return useMemo(() => {
    let soakTimes: Record<string, Duration> = {};

    if (
      actionContext?.action?.type !== IAction.PROMOTE &&
      actionContext?.action?.type !== IAction.PROMOTE_DOWNSTREAM
    ) {
      return soakTimes;
    }

    const currentlyPromotingStages: string[] = [];

    if (actionContext?.action?.type === IAction.PROMOTE) {
      currentlyPromotingStages.push(actionContext?.action?.stage?.metadata?.name || '');
    } else {
      const subscribers =
        dictionaryContext?.subscribersByStage?.[actionContext?.action?.stage?.metadata?.name || ''];

      for (const subscriber of subscribers || []) {
        currentlyPromotingStages.push(subscriber);
      }
    }

    for (const currentlyPromotingStage of currentlyPromotingStages) {
      soakTimes = soakTimeForPromotingStage({
        stage: dictionaryContext?.stageByName?.[currentlyPromotingStage] as Stage,
        freights,
        soakTimesByFreight: soakTimes,
        stageByName: dictionaryContext?.stageByName as Record<string, Stage>,
        subscribersByStage: dictionaryContext?.subscribersByStage as Record<string, Set<string>>
      });
    }

    return soakTimes;
  }, [actionContext, freights, dictionaryContext]);
};
