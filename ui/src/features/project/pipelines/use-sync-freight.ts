import { useEffect } from 'react';

import { queryCache } from '@ui/features/utils/cache';
import { Freight, Stage } from '@ui/gen/api/v2/models';

export const useSyncFreight = (payload: {
  project: string;
  freights?: Record<string, Freight>;
  freightInStages?: Record<string, Stage[]>;
}) => {
  useEffect(() => {
    if (payload.freights && payload.freightInStages) {
      const freights = Object.keys(payload.freights || {});
      const freightInStages = Object.keys(payload.freightInStages || {});

      for (const freightInStage of freightInStages) {
        if (!freights.find((f) => f === freightInStage)) {
          queryCache.freight.refetchQueryFreight(payload.project);
          return;
        }
      }
    }
  }, [payload]);
};
