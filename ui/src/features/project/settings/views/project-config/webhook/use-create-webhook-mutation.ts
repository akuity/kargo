import { useMutation as useReactQueryMutation, useQueryClient } from '@tanstack/react-query';
import { notification } from 'antd';
import { parse, stringify } from 'yaml';

import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';
import {
  useCreateProjectGenericCredentials,
  useDeleteProjectGenericCredentials
} from '@ui/gen/api/v2/credentials/credentials';
import { useCreateResource, useUpdateResource } from '@ui/gen/api/v2/resources/resources';

type ProjectConfigPartial = {
  apiVersion?: string;
  kind?: string;
  metadata?: { name?: string; namespace?: string };
  spec?: {
    webhookReceivers?: Array<Record<string, unknown>>;
  };
};

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
  const queryClient = useQueryClient();
  const createResourceMutation = useCreateResource();
  const updateResourceMutation = useUpdateResource();
  const createProjectSecretMutation = useCreateProjectGenericCredentials();
  const deleteProjectSecretMutation = useDeleteProjectGenericCredentials();

  return useReactQueryMutation({
    mutationFn: async (payload: createWebhookPayload) => {
      await createProjectSecretMutation.mutateAsync({
        project: payload.secret.namespace,
        data: {
          name: payload.secret.name,
          data: payload.secret.data
        }
      });

      let projectConfig: ProjectConfigPartial = parse(payload.projectConfigYAML);

      if (payload.projectConfigYAML === '') {
        projectConfig = {
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

      const resourceMutation =
        payload.projectConfigYAML === '' ? createResourceMutation : updateResourceMutation;

      try {
        await resourceMutation.mutateAsync({ data: stringify(projectConfig) });
      } catch {
        await deleteProjectSecretMutation.mutateAsync({
          project: payload.secret.namespace,
          genericCredentials: payload.secret.name
        });
      }
    },
    onSuccess: (_, vars) => {
      notification.success({
        message: `Successfully added webhook for ${vars.webhookReceiver}`,
        placement: 'bottomRight'
      });
      queryClient.invalidateQueries({
        queryKey: getGetProjectConfigQueryKey(vars.secret.namespace)
      });
      opts?.onSuccess?.();
    }
  });
};
