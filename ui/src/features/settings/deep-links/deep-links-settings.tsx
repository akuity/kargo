import { DeepLinks } from '@ui/features/common/settings/deep-links/deep-links';
import { useSaveClusterDeepLinks } from '@ui/features/common/settings/deep-links/use-save-deep-links';
import { useGetClusterConfig } from '@ui/gen/api/v2/system/system';
import { ApiError } from '@ui/lib/api/custom-fetch';

export const DeepLinksSettings = () => {
  const getClusterConfigQuery = useGetClusterConfig({ query: { meta: { silent404: true } } });

  const clusterConfig = getClusterConfigQuery.data?.data;

  // a missing ClusterConfig is expected -- saving creates it. Any other failure
  // leaves the stored spec unknown, and a save would overwrite all of it
  const { error } = getClusterConfigQuery;
  const loadFailed = !!error && !(error instanceof ApiError && error.isNotFound());

  const { mutateAsync: onUpdate } = useSaveClusterDeepLinks();

  return (
    <DeepLinks
      scope='cluster'
      freightLinks={clusterConfig?.spec?.freightLinks ?? []}
      stageLinks={clusterConfig?.spec?.stageLinks ?? []}
      loading={getClusterConfigQuery.isLoading}
      loadFailed={loadFailed}
      onUpdate={onUpdate}
    />
  );
};
