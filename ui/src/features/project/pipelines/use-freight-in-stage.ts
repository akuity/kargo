import { useMemo } from 'react';

import { getCurrentFreight } from '@ui/features/common/utils';
import { Stage } from '@ui/gen/api/v2/models';

export const useFreightInStage = (stages: Stage[]): Record<string, Stage[]> =>
  useMemo(() => {
    const freightInStage: Record<string, Stage[]> = {};
    for (const stage of stages) {
      const currentFreights = getCurrentFreight(stage);

      for (const currentFreight of currentFreights) {
        if (!currentFreight.name) {
          continue;
        }

        if (!freightInStage[currentFreight.name]) {
          freightInStage[currentFreight.name] = [];
        }

        freightInStage[currentFreight.name].push(stage);
      }
    }
    return freightInStage;
  }, [stages]);
