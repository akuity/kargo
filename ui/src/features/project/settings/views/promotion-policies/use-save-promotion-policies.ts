import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import {
  configurationForUpdate,
  configurationMutationScope,
  updateConfigurationCache
} from '@ui/features/common/settings/configuration-cache';
import { projectConfigGen } from '@ui/features/promotion-windows/utils';
import { ProjectConfig, PromotionPolicy } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

export const useSaveProjectPromotionPolicies = (project: string, projectConfig?: ProjectConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    scope: configurationMutationScope(project),
    mutationFn: async (promotionPolicies: PromotionPolicy[]) => {
      const current = (await configurationForUpdate(queryClient, project)) ?? projectConfig;
      return updateResource.mutateAsync({
        data: projectConfigGen.v1alpha1({
          metadata: {
            name: project,
            namespace: project,
            ...current?.metadata
          },
          spec: {
            ...current?.spec,
            promotionPolicies
          }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, project, response.data.results);
      message.success({ content: 'Promotion policies updated' });
    }
  });
};
