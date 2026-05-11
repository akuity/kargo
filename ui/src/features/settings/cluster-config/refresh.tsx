import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { queryCache } from '@ui/features/utils/cache';
import { useRefreshClusterConfig } from '@ui/gen/api/v2/system/system';

export const Refresh = () => {
  const refreshMutation = useRefreshClusterConfig({
    mutation: {
      onSuccess: () => queryCache.clusterConfig.refetch()
    }
  });

  return (
    <Tooltip title='Rotated webhook secrets? Refresh ClusterConfig to generate the new webhook URL.'>
      <Button
        icon={<FontAwesomeIcon icon={faUndo} />}
        loading={refreshMutation.isPending}
        onClick={() => refreshMutation.mutate()}
      >
        Refresh
      </Button>
    </Tooltip>
  );
};
