import { useMemo } from 'react';

import { Stage } from '@ui/gen/api/v2/models';

export const useStageByName = (stages: Stage[]): Record<string, Stage> =>
  useMemo(() => {
    const stageByName: Record<string, Stage> = {};

    for (const stage of stages) {
      const stageName = stage?.metadata?.name || '';

      stageByName[stageName] = stage;
    }

    return stageByName;
  }, [stages]);
