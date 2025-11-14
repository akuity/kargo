import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const useStageAutoPromotionMap = (stages: Stage[]): Record<string, boolean> =>
  useMemo(() => {
    const map: Record<string, boolean> = {};

    for (const stage of stages) {
      if (stage?.status?.autoPromotionEnabled) {
        map[stage?.metadata?.name || ''] = true;
      }
    }
    return map;
  }, [stages]);
