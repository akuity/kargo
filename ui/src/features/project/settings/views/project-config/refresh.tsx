import { useMutation } from '@connectrpc/connect-query';
import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { queryCache } from '@ui/features/utils/cache';
import { refreshResource } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

export const Refresh = (props: { project: string }) => {
  const refreshResourceTypeProjectConfig = 'ProjectConfig';
  const refreshResourceMutation = useMutation(refreshResource, {
    onSuccess: () => queryCache.projectConfig.refetch()
  });

  return (
    <Tooltip title='Rotated webhook secrets? Refresh ProjectConfig to generate the new webhook URL.'>
      <Button
        icon={<FontAwesomeIcon icon={faUndo} />}
        loading={refreshResourceMutation.isPending}
        onClick={() =>
          refreshResourceMutation.mutate({
            project: props.project,
            resourceType: refreshResourceTypeProjectConfig
          })
        }
      >
        Refresh
      </Button>
    </Tooltip>
  );
};
