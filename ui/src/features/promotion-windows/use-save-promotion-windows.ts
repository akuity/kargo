import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';
import { ClusterConfig, ProjectConfig, PromotionWindow } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import { getGetClusterConfigQueryKey } from '@ui/gen/api/v2/system/system';

import { clusterConfigGen, projectConfigGen } from './utils';

export const useSaveProjectPromotionWindows = (project: string, projectConfig?: ProjectConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: (promotionWindows: PromotionWindow[]) => {
      return updateResource.mutateAsync({
        data: projectConfigGen.v1alpha1({
          metadata: {
            name: project,
            namespace: project,
            ...projectConfig?.metadata
          },
          spec: {
            ...projectConfig?.spec,
            promotionWindows
          }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: () => {
      message.success({ content: 'Promotion windows updated' });
      queryClient.invalidateQueries({ queryKey: getGetProjectConfigQueryKey(project) });
    }
  });
};

export const useSaveClusterPromotionWindows = (clusterConfig?: ClusterConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: (promotionWindows: PromotionWindow[]) => {
      return updateResource.mutateAsync({
        data: clusterConfigGen.v1alpha1({
          metadata: { name: 'cluster', ...clusterConfig?.metadata },
          spec: { ...clusterConfig?.spec, promotionWindows }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: () => {
      message.success({ content: 'Promotion windows updated' });
      queryClient.invalidateQueries({ queryKey: getGetClusterConfigQueryKey() });
    }
  });
};
