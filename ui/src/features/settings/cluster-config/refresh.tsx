import { useMutation } from '@connectrpc/connect-query';
import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { queryCache } from '@ui/features/utils/cache';
import { refreshClusterConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const Refresh = () => {
  const refreshClusterConfigMutation = useMutation(refreshClusterConfig, {
    onSuccess: () => queryCache.clusterConfig.refetch()
  });

  return (
    <Tooltip title='Rotated webhook secrets? Refresh ClusterConfig to generate the new webhook URL.'>
      <Button
        icon={<FontAwesomeIcon icon={faUndo} />}
        loading={refreshClusterConfigMutation.isPending}
        onClick={() => refreshClusterConfigMutation.mutate({})}
      >
        Refresh
      </Button>
    </Tooltip>
  );
};
