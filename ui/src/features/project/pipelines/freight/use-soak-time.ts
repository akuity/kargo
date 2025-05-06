import { useMemo } from 'react';

import { useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';
import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

import { getSoakTime } from './get-soak-time';

export const useSoakTime = (freights: Freight[]) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  return useMemo(() => {
    const soakTimes: Record<string, string> = {};

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

      const soakTimesForFreight = [];
      for (const stage of freightInStages) {
        const stageDetails = dictionaryContext?.stageByName?.[stage];

        if (stageDetails && requireSoakTime) {
          getSoakTime({
            freight,
            freightInStage: dictionaryContext?.stageByName?.[stage],
            requiredSoakTime: requireSoakTime
          });
        }
      }
    }
  }, [actionContext, freights, dictionaryContext]);
};
