import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const usePromotionHistory = (stages: Stage[]) => {
  return useMemo(() => {
    const history: Record<string, Record<string, Record<string, number[]>>> = {};

    for (const stage of stages) {
      const stageName = stage.metadata?.name || '';
      if (!stageName) return history;

      stage.status?.freightHistory?.forEach((freightGroup, freightIndex) => {
        for (const freightRef of Object.values(freightGroup.items)) {
          for (const image of freightRef.images) {
            const repoURL = image.repoURL || '';
            const tag = image.tag || '';

            if (!history[repoURL]) {
              history[repoURL] = {};
            }
            if (!history[repoURL][tag]) {
              history[repoURL][tag] = {};
            }
            if (!history[repoURL][tag][stageName]) {
              history[repoURL][tag][stageName] = [];
            }

            history[repoURL][tag][stageName].push(freightIndex + 1);
          }
        }
      });
    }

    return history;
  }, [stages]);
};
