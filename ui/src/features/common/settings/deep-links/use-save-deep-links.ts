import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { clusterConfigGen, projectConfigGen } from '@ui/features/promotion-windows/utils';
import { getGetProjectConfigQueryOptions } from '@ui/gen/api/v2/core/core';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import { getGetClusterConfigQueryOptions } from '@ui/gen/api/v2/system/system';
import { ApiError } from '@ui/lib/api/custom-fetch';

import { DeepLinkPatch } from './deep-links';

/**
 * A save sends the whole spec, so it has to be built on the current one. The
 * copy the page rendered from can already be a save behind, and the server
 * takes whatever it is sent, so the config is re-read here and the patch
 * merged onto that. A config that does not exist yet is not an error: saving
 * creates it.
 */
const latestConfig = async <T>(fetch: () => Promise<{ data: T }>): Promise<T | undefined> => {
  try {
    return (await fetch()).data;
  } catch (error) {
    if (error instanceof ApiError && error.isNotFound()) {
      return undefined;
    }
    throw error;
  }
};

export const useSaveProjectDeepLinks = (project: string) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: async (patch: DeepLinkPatch) => {
      const projectConfig = await latestConfig(() =>
        queryClient.fetchQuery(
          getGetProjectConfigQueryOptions(project, { query: { meta: { silent404: true } } })
        )
      );

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
    onSuccess: () => {
      message.success({ content: 'Deep links updated' });
      queryClient.invalidateQueries({
        queryKey: getGetProjectConfigQueryOptions(project).queryKey
      });
    }
  });
};

export const useSaveClusterDeepLinks = () => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: async (patch: DeepLinkPatch) => {
      const clusterConfig = await latestConfig(() =>
        queryClient.fetchQuery(
          getGetClusterConfigQueryOptions({ query: { meta: { silent404: true } } })
        )
      );

      return updateResource.mutateAsync({
        data: clusterConfigGen.v1alpha1({
          metadata: { name: 'cluster', ...clusterConfig?.metadata },
          spec: { ...clusterConfig?.spec, ...patch }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: () => {
      message.success({ content: 'Deep links updated' });
      queryClient.invalidateQueries({ queryKey: getGetClusterConfigQueryOptions().queryKey });
    }
  });
};
