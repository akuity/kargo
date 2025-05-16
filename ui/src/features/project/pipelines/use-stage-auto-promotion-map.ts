import { useMemo } from 'react';

import { Project, ProjectConfig } from '@ui/gen/api/v1alpha1/generated_pb';

export const useStageAutoPromotionMap = (
  project: Project,
  projectConfig: ProjectConfig
): Record<string, boolean> =>
  useMemo(() => {
    const map: Record<string, boolean> = {};

    // deprecated
    for (const policy of project?.spec?.promotionPolicies || []) {
      if (policy.stage) {
        map[policy.stage] = policy.autoPromotionEnabled;
      }
    }

    for (const policy of projectConfig?.spec?.promotionPolicies || []) {
      if (policy.stage) {
        map[policy.stage] = policy.autoPromotionEnabled;
      }

      if (policy.stageSelector?.name) {
        map[policy.stageSelector?.name || ''] = policy.autoPromotionEnabled;
      }
    }

    return map;
  }, [project, projectConfig]);
