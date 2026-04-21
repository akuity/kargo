import { faUndo } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Tooltip } from 'antd';

import { queryCache } from '@ui/features/utils/cache';
import { useRefreshProjectConfig } from '@ui/gen/api/v2/core/core';

export const Refresh = (props: { project: string }) => {
  const refreshProjectConfigMutation = useRefreshProjectConfig({
    mutation: {
      onSuccess: () => queryCache.projectConfig.refetch()
    }
  });

  return (
    <Tooltip title='Rotated webhook secrets? Refresh ProjectConfig to generate the new webhook URL.'>
      <Button
        icon={<FontAwesomeIcon icon={faUndo} />}
        loading={refreshProjectConfigMutation.isPending}
        onClick={() => refreshProjectConfigMutation.mutate({ project: props.project })}
      >
        Refresh
      </Button>
    </Tooltip>
  );
};
