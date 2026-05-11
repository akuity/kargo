import { useMutation as useReactQueryMutation } from '@tanstack/react-query';
import { notification } from 'antd';
import { parse, stringify } from 'yaml';

import { queryCache } from '@ui/features/utils/cache';
import {
  useCreateSystemGenericCredentials,
  useDeleteSystemGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { useCreateResource, useUpdateResource } from '@ui/gen/api/v2/resources/resources';

type ClusterConfigPartial = {
  apiVersion?: string;
  kind?: string;
  metadata?: { name?: string };
  spec?: {
    webhookReceivers?: Array<Record<string, unknown>>;
  };
};

type createWebhookPayload = {
  clusterConfigYAML: string;
  webhookReceiver: string;
  webhookReceiverName: string;
  secret: {
    name: string;
    data: Record<string, string>;
  };
};

export const useCreateWebhookMutation = (opts?: { onSuccess?: () => void }) => {
  const createResourceMutation = useCreateResource();
  const updateResourceMutation = useUpdateResource();
  const createSystemSecretMutation = useCreateSystemGenericCredentials();
  const deleteSystemSecretMutation = useDeleteSystemGenericCredentials();

  return useReactQueryMutation({
    mutationFn: async (payload: createWebhookPayload) => {
      await createSystemSecretMutation.mutateAsync({
        data: {
          name: payload.secret.name,
          data: payload.secret.data
        }
      });

      try {
        let clusterConfig: ClusterConfigPartial = parse(payload.clusterConfigYAML);

        if (payload.clusterConfigYAML === '') {
          clusterConfig = {
            apiVersion: 'kargo.akuity.io/v1alpha1',
            kind: 'ClusterConfig',
            metadata: {
              name: 'cluster'
            },
            spec: {
              webhookReceivers: []
            }
          };
        }

        if (!clusterConfig.spec) {
          clusterConfig.spec = {
            webhookReceivers: []
          };
        }

        if (!clusterConfig.spec?.webhookReceivers?.length) {
          clusterConfig.spec.webhookReceivers = [];
        }

        clusterConfig.spec?.webhookReceivers.push({
          name: payload.webhookReceiverName,
          [payload.webhookReceiver]: {
            secretRef: {
              name: payload.secret.name
            }
          }
        });

        const resourceMutation =
          payload.clusterConfigYAML === '' ? createResourceMutation : updateResourceMutation;
        await resourceMutation.mutateAsync({ data: stringify(clusterConfig) });
      } catch (e) {
        await deleteSystemSecretMutation.mutateAsync({
          genericCredentials: payload.secret.name
        });

        throw e;
      }
    },
    onSuccess: (_, vars) => {
      notification.success({
        message: `Successfully added webhook for ${vars.webhookReceiver}`,
        placement: 'bottomRight'
      });
      queryCache.clusterConfig?.refetch();
      opts?.onSuccess?.();
    },
    onError: (err) =>
      notification.error({
        message: (err as Error).message || 'Failed to create webhook',
        placement: 'bottomRight'
      })
  });
};
