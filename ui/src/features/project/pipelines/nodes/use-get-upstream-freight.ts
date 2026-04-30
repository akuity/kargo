import { useMemo } from 'react';

import { getCurrentFreight } from '@ui/features/common/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { useDictionaryContext } from '../context/dictionary-context';

export const useGetUpstreamFreight = (stage: Stage) => {
  const dictionaryContext = useDictionaryContext();

  return useMemo(() => {
    const subscribersByStage = dictionaryContext?.subscribersByStage;

    const stageName = stage?.metadata?.name || '';

    const upstreamStages = [];

    for (const [source, dests] of Object.entries(subscribersByStage || {})) {
      if (dests.has(stageName)) {
        upstreamStages.push(dictionaryContext?.stageByName[source]);
      }
    }

    // feature: select from multiple stages
    if (upstreamStages.length !== 1) {
      return null;
    }

    if (!upstreamStages[0]) {
      return null;
    }

    return getCurrentFreight(upstreamStages[0]);
  }, [dictionaryContext?.subscribersByStage, dictionaryContext?.stageByName, stage]);
};
