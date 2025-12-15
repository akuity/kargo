import { useMutation } from '@connectrpc/connect-query';
import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { queryCache } from '@ui/features/utils/cache';
import { refreshResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const Refresh = () => {
  const refreshResourceTypeClusterConfig = 'ClusterConfig';
  const refreshResourceMutation = useMutation(refreshResource, {
    onSuccess: () => queryCache.clusterConfig.refetch()
  });

  return (
    <Tooltip title='Rotated webhook secrets? Refresh ClusterConfig to generate the new webhook URL.'>
      <Button
        icon={<FontAwesomeIcon icon={faUndo} />}
        loading={refreshResourceMutation.isPending}
        onClick={() =>
          refreshResourceMutation.mutate({
            resourceType: refreshResourceTypeClusterConfig
          })
        }
      >
        Refresh
      </Button>
    </Tooltip>
  );
};
