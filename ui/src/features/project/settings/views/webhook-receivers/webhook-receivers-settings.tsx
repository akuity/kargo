import { useParams } from 'react-router-dom';

import { useSaveProjectWebhookReceivers } from '@ui/features/common/settings/webhook-receivers/use-save-webhook-receivers';
import { useProjectWebhookSecrets } from '@ui/features/common/settings/webhook-receivers/use-webhook-secrets';
import { WebhookReceivers } from '@ui/features/common/settings/webhook-receivers/webhook-receivers';
import { useGetProjectConfig } from '@ui/gen/api/v2/core/core';
import { ApiError } from '@ui/lib/api/custom-fetch';

import { Refresh } from '../project-config/refresh';

export const WebhookReceiversSettings = () => {
  const { name = '' } = useParams();

  const getProjectConfigQuery = useGetProjectConfig(name, {
    query: { meta: { silent404: true } }
  });

  const projectConfig = getProjectConfigQuery.data?.data;

  // a missing ProjectConfig is expected -- saving creates it. Any other failure
  // leaves the stored spec unknown, and a save would overwrite all of it
  const { error } = getProjectConfigQuery;
  const loadFailed = !!error && !(error instanceof ApiError && error.isNotFound());

  const secrets = useProjectWebhookSecrets(name);
  const saveMutation = useSaveProjectWebhookReceivers(name, projectConfig);

  return (
    <WebhookReceivers
      webhookReceivers={projectConfig?.spec?.webhookReceivers ?? []}
      receiverDetails={projectConfig?.status?.webhookReceivers ?? []}
      secrets={secrets}
      loading={getProjectConfigQuery.isLoading}
      loadFailed={loadFailed}
      onUpdate={saveMutation.mutateAsync}
      refresh={<Refresh project={name} />}
    />
  );
};
