import { useMemo } from 'react';

import { Project } from '@ui/gen/api/v1alpha1/generated_pb';

export const useStageAutoPromotionMap = (project: Project): Record<string, boolean> =>
  useMemo(() => {
    const map: Record<string, boolean> = {};

    for (const policy of project?.spec?.promotionPolicies || []) {
      if (policy.stage) {
        map[policy.stage] = policy.autoPromotionEnabled;
      }
    }

    return map;
  }, [project]);
