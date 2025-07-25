import { useMemo } from 'react';

import { Project, Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const useStageAutoPromotionMap = (
  project: Project,
  stages: Stage[]
): Record<string, boolean> =>
  useMemo(() => {
    const map: Record<string, boolean> = {};

    // deprecated
    for (const policy of project?.spec?.promotionPolicies || []) {
      if (policy.stage) {
        map[policy.stage] = policy.autoPromotionEnabled;
      }
    }

    for (const stage of stages) {
      if (stage?.status?.autoPromotionEnabled) {
        map[stage?.metadata?.name || ''] = true;
      }
    }
    return map;
  }, [project, stages]);
