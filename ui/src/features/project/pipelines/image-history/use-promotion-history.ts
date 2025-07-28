import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const usePromotionHistory = (stages: Stage[]) => {
  return useMemo(() => {
    const history: Record<string, Record<string, Record<string, number[]>>> = {};

    stages.forEach((stage) => {
      const stageName = stage.metadata?.name || '';
      if (!stageName) return;

      stage.status?.freightHistory?.forEach((freightGroup, freightIndex) => {
        Object.values(freightGroup.items || {}).forEach((freightRef) => {
          freightRef.images?.forEach((image) => {
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
          });
        });
      });
    });

    return history;
  }, [stages]);
};
