import { Duration, milliseconds } from 'date-fns';
import { useMemo } from 'react';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Freight, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { getSoakTime } from './get-soak-time';

/**
 * soak time calculation:
 *
 * when pipeline is in promotion mode, we will calculate soak time for each freight
 *
 * first find if currently promoting stage has required soak time from upstream stage(s)
 * if it is then for each promotion ineligible freight, find how much time it is in upstream stage(s) of currently promoting stage
 * if there are multiple upstream stages then find longest time since the freight is in one of stage
 * then compare it with currently promoting stages required soak time
 *
 * above process is for "promote to stage"
 * for promote to downstream, we have to calculate for each downstream stages required soak time and pick the longest one
 */

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

    // there will be multiple soak times due to the fact that there are potentially multiple source
    // in that case we just need to find longest soak time
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
        milliseconds(curr) > milliseconds(max) ? curr : max
      );

      if (!payload.soakTimesByFreight[freightName]) {
        payload.soakTimesByFreight[freightName] = maxSoakTime;
      } else {
        // if we find soak time already calculated for that freight
        // that means we are looking for "downstream" promotion mode
        const existingSoakTime = payload.soakTimesByFreight[freightName];
        payload.soakTimesByFreight[freightName] =
          milliseconds(existingSoakTime) > milliseconds(maxSoakTime)
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

    // for normal promotion mode, currently promoting stage is selected stage
    // for downstream, it is subscribers of the selected stage
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
