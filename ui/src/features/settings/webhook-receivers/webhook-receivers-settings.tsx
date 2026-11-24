import { useSaveClusterWebhookReceivers } from '@ui/features/common/settings/webhook-receivers/use-save-webhook-receivers';
import { useClusterWebhookSecrets } from '@ui/features/common/settings/webhook-receivers/use-webhook-secrets';
import { WebhookReceivers } from '@ui/features/common/settings/webhook-receivers/webhook-receivers';
import { useGetClusterConfig } from '@ui/gen/api/v2/system/system';
import { ApiError } from '@ui/lib/api/custom-fetch';

import { Refresh } from '../cluster-config/refresh';

export const WebhookReceiversSettings = () => {
  const getClusterConfigQuery = useGetClusterConfig({
    query: { meta: { silent404: true } }
  });

  const clusterConfig = getClusterConfigQuery.data?.data;

  // a missing ClusterConfig is expected -- saving creates it. Any other failure
  // leaves the stored spec unknown, and a save would overwrite all of it
  const { error } = getClusterConfigQuery;
  const loadFailed = !!error && !(error instanceof ApiError && error.isNotFound());

  const secrets = useClusterWebhookSecrets();
  const saveMutation = useSaveClusterWebhookReceivers(clusterConfig);

  return (
    <WebhookReceivers
      webhookReceivers={clusterConfig?.spec?.webhookReceivers ?? []}
      receiverDetails={clusterConfig?.status?.webhookReceivers ?? []}
      secrets={secrets}
      loading={getClusterConfigQuery.isLoading}
      loadFailed={loadFailed}
      onUpdate={saveMutation.mutateAsync}
      refresh={<Refresh />}
    />
  );
};
