import { useMutation } from '@connectrpc/connect-query';
import { useMutation as useReactQueryMutation } from '@tanstack/react-query';
import { notification } from 'antd';
import { parse, stringify } from 'yaml';

import { queryCache } from '@ui/features/utils/cache';
import {
  createOrUpdateResource,
  createGenericCredentials,
  deleteGenericCredentials
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ProjectConfig } from '@ui/gen/api/v1alpha1/generated_pb';
import { PartialRecursive } from '@ui/utils/connectrpc-utils';

type createWebhookPayload = {
  projectConfigYAML: string;
  webhookReceiver: string;
  webhookReceiverName: string;
  secret: {
    namespace: string;
    name: string;
    data: Record<string, string>;
  };
};

export const useCreateWebhookMutation = (opts?: { onSuccess?: () => void }) => {
  const createOrUpdateMutation = useMutation(createOrUpdateResource);
  const createProjectSecretMutation = useMutation(createGenericCredentials);
  const deleteProjectSecretMutation = useMutation(deleteGenericCredentials);

  return useReactQueryMutation({
    mutationFn: async (payload: createWebhookPayload) => {
      await createProjectSecretMutation.mutateAsync({
        project: payload.secret.namespace,
        name: payload.secret.name,
        data: payload.secret.data
      });

      let projectConfig = parse(payload.projectConfigYAML) as PartialRecursive<ProjectConfig>;

      if (payload.projectConfigYAML === '') {
        projectConfig = {
          // @ts-expect-error apiVersion required when creating resource
          apiVersion: 'kargo.akuity.io/v1alpha1',
          kind: 'ProjectConfig',
          metadata: {
            name: payload.secret.namespace,
            namespace: payload.secret.namespace
          },
          spec: {
            webhookReceivers: []
          }
        };
      }

      if (!projectConfig.spec) {
        projectConfig.spec = {
          webhookReceivers: []
        };
      }

      if (!projectConfig.spec?.webhookReceivers?.length) {
        projectConfig.spec.webhookReceivers = [];
      }

      projectConfig.spec?.webhookReceivers.push({
        name: payload.webhookReceiverName,
        [payload.webhookReceiver]: {
          secretRef: {
            name: payload.secret.name
          }
        }
      });

      const textEncoder = new TextEncoder();

      try {
        await createOrUpdateMutation.mutateAsync({
          manifest: textEncoder.encode(stringify(projectConfig))
        });
      } catch {
        await deleteProjectSecretMutation.mutateAsync({
          name: payload.secret.name,
          project: payload.secret.namespace
        });
      }
    },
    onSuccess: (_, vars) => {
      notification.success({
        message: `Successfully added webhook for ${vars.webhookReceiver}`,
        placement: 'bottomRight'
      });
      queryCache.projectConfig.refetch();
      opts?.onSuccess?.();
    }
  });
};
