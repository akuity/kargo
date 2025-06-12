import { useMutation } from '@connectrpc/connect-query';
import { useMutation as useReactQueryMutation } from '@tanstack/react-query';
import { notification } from 'antd';
import { parse, stringify } from 'yaml';

import { queryCache } from '@ui/features/utils/cache';
import {
  createOrUpdateResource,
  createProjectSecret,
  deleteProjectSecret
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { ProjectConfig } from '@ui/gen/api/v1alpha1/generated_pb';

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
  const createProjectSecretMutation = useMutation(createProjectSecret);
  const deleteProjectSecretMutation = useMutation(deleteProjectSecret);

  return useReactQueryMutation({
    mutationFn: async (payload: createWebhookPayload) => {
      await createProjectSecretMutation.mutateAsync({
        project: payload.secret.namespace,
        name: payload.secret.name,
        data: payload.secret.data
      });

      let projectConfig = parse(payload.projectConfigYAML) as ProjectConfig;

      if (payload.projectConfigYAML === '') {
        projectConfig = {
          apiVersion: 'kargo.akuity.io/v1alpha1',
          kind: 'ProjectConfig',
          // @ts-expect-error expected
          metadata: {
            name: payload.secret.namespace,
            namespace: payload.secret.namespace
          },
          spec: {
            // @ts-expect-error expected
            webhookReceivers: []
          }
        };
      }

      if (!projectConfig.spec) {
        projectConfig.spec = {
          // @ts-expect-error expected
          webhookReceivers: []
        };
      }

      // @ts-expect-error expected
      if (!projectConfig.spec?.webhookReceivers?.length) {
        // @ts-expect-error expected
        projectConfig.spec.webhookReceivers = [];
      }

      // @ts-expect-error expected
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
