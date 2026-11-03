import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { clusterConfigGen, projectConfigGen } from '@ui/features/promotion-windows/utils';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

import {
  configurationForUpdate,
  configurationMutationScope,
  updateConfigurationCache
} from '../configuration-cache';

import { DeepLinkPatch } from './deep-links';

export const useSaveProjectDeepLinks = (project: string) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    scope: configurationMutationScope(project),
    mutationFn: async (patch: DeepLinkPatch) => {
      const projectConfig = await configurationForUpdate(queryClient, project);

      return updateResource.mutateAsync({
        data: projectConfigGen.v1alpha1({
          metadata: {
            name: project,
            namespace: project,
            ...projectConfig?.metadata
          },
          spec: {
            ...projectConfig?.spec,
            ...patch
          }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, project, response.data.results);
      message.success({ content: 'Deep links updated' });
    }
  });
};

export const useSaveClusterDeepLinks = () => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    scope: configurationMutationScope(undefined),
    mutationFn: async (patch: DeepLinkPatch) => {
      const clusterConfig = await configurationForUpdate(queryClient, undefined);

      return updateResource.mutateAsync({
        data: clusterConfigGen.v1alpha1({
          metadata: { name: 'cluster', ...clusterConfig?.metadata },
          spec: { ...clusterConfig?.spec, ...patch }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, undefined, response.data.results);
      message.success({ content: 'Deep links updated' });
    }
  });
};
