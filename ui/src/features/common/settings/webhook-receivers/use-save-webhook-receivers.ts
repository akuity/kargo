import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { clusterConfigGen, projectConfigGen } from '@ui/features/promotion-windows/utils';
import { ClusterConfig, ProjectConfig, WebhookReceiverConfig } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

import {
  configurationForUpdate,
  configurationMutationScope,
  updateConfigurationCache
} from '../configuration-cache';

const savedMessage = 'Webhook receivers updated';

export const useSaveProjectWebhookReceivers = (project: string, projectConfig?: ProjectConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    scope: configurationMutationScope(project),
    mutationFn: async (webhookReceivers: WebhookReceiverConfig[]) => {
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
            webhookReceivers
          }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, project, response.data.results);
      message.success({ content: savedMessage });
    }
  });
};

export const useSaveClusterWebhookReceivers = (clusterConfig?: ClusterConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    scope: configurationMutationScope(undefined),
    mutationFn: async (webhookReceivers: WebhookReceiverConfig[]) => {
      const current = (await configurationForUpdate(queryClient, undefined)) ?? clusterConfig;
      return updateResource.mutateAsync({
        data: clusterConfigGen.v1alpha1({
          metadata: { name: 'cluster', ...current?.metadata },
          spec: { ...current?.spec, webhookReceivers }
        }),
        params: { upsert: true }
      });
    },
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, undefined, response.data.results);
      message.success({ content: savedMessage });
    }
  });
};
