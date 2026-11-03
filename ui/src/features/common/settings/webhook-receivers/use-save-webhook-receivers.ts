import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { clusterConfigGen, projectConfigGen } from '@ui/features/promotion-windows/utils';
import { ClusterConfig, ProjectConfig, WebhookReceiverConfig } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';

import { updateConfigurationCache } from '../configuration-cache';

const savedMessage = 'Webhook receivers updated';

export const useSaveProjectWebhookReceivers = (project: string, projectConfig?: ProjectConfig) => {
  const queryClient = useQueryClient();
  const updateResource = useUpdateResource();

  return useMutation({
    mutationFn: (webhookReceivers: WebhookReceiverConfig[]) =>
      updateResource.mutateAsync({
        data: projectConfigGen.v1alpha1({
          metadata: {
            name: project,
            namespace: project,
            ...projectConfig?.metadata
          },
          spec: {
            ...projectConfig?.spec,
            webhookReceivers
          }
        }),
        params: { upsert: true }
      }),
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
    mutationFn: (webhookReceivers: WebhookReceiverConfig[]) =>
      updateResource.mutateAsync({
        data: clusterConfigGen.v1alpha1({
          metadata: { name: 'cluster', ...clusterConfig?.metadata },
          spec: { ...clusterConfig?.spec, webhookReceivers }
        }),
        params: { upsert: true }
      }),
    onSuccess: async (response) => {
      await updateConfigurationCache(queryClient, undefined, response.data.results);
      message.success({ content: savedMessage });
    }
  });
};
