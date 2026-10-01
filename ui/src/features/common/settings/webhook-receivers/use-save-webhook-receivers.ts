import { useMutation, useQueryClient } from '@tanstack/react-query';
import { message } from 'antd';

import { clusterConfigGen, projectConfigGen } from '@ui/features/promotion-windows/utils';
import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';
import { ClusterConfig, ProjectConfig, WebhookReceiverConfig } from '@ui/gen/api/v2/models';
import { useUpdateResource } from '@ui/gen/api/v2/resources/resources';
import { getGetClusterConfigQueryKey } from '@ui/gen/api/v2/system/system';

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
    onSuccess: () => {
      message.success({ content: savedMessage });
      queryClient.invalidateQueries({ queryKey: getGetProjectConfigQueryKey(project) });
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
    onSuccess: () => {
      message.success({ content: savedMessage });
      queryClient.invalidateQueries({ queryKey: getGetClusterConfigQueryKey() });
    }
  });
};
