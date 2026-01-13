import { useMutation } from '@connectrpc/connect-query';
import { useMutation as useReactQueryMutation } from '@tanstack/react-query';
import { notification } from 'antd';
import { parse, stringify } from 'yaml';

import { queryCache } from '@ui/features/utils/cache';
import {
  createGenericCredentials,
  createOrUpdateResource,
  deleteGenericCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ClusterConfig } from '@ui/gen/api/v1alpha1/generated_pb';
import { PartialRecursive } from '@ui/utils/connectrpc-utils';

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
  const createOrUpdateMutation = useMutation(createOrUpdateResource);
  const createSystemSecretMutation = useMutation(createGenericCredentials);
  const deleteSystemSecretMutation = useMutation(deleteGenericCredentials);

  return useReactQueryMutation({
    mutationFn: async (payload: createWebhookPayload) => {
      await createSystemSecretMutation.mutateAsync({
        systemLevel: true,
        name: payload.secret.name,
        data: payload.secret.data
      });

      try {
        let clusterConfig = parse(payload.clusterConfigYAML) as PartialRecursive<ClusterConfig>;

        if (payload.clusterConfigYAML === '') {
          clusterConfig = {
            // @ts-expect-error apiVersion required when creating resource
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

        const textEncoder = new TextEncoder();

        await createOrUpdateMutation.mutateAsync({
          manifest: textEncoder.encode(stringify(clusterConfig))
        });
      } catch (e) {
        await deleteSystemSecretMutation.mutateAsync({
          systemLevel: true,
          name: payload.secret.name
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
