import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { projectConfigGen } from '@ui/features/promotion-windows/utils';
import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';
import { ProjectConfig, PromotionPolicy } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

export const useSaveProjectPromotionPolicies = (project: string, projectConfig?: ProjectConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: (promotionPolicies: PromotionPolicy[]) =>
      updateResource.mutateAsync({
        data: projectConfigGen.v1alpha1({
          metadata: {
            name: project,
            namespace: project,
            ...projectConfig?.metadata
          },
          spec: {
            ...projectConfig?.spec,
            promotionPolicies
          }
        }),
        params: { upsert: true }
      }),
    onSuccess: () => {
      message.success({ content: 'Promotion policies updated' });
      queryClient.invalidateQueries({ queryKey: getGetProjectConfigQueryKey(project) });
    }
  });
};
