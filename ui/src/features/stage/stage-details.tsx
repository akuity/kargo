import { useQuery } from '@tanstack/react-query';
import { Divider, Drawer, Empty, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status-icon/health-status-icon';
import { AvailableStates } from '@ui/features/stage/available-states';
import { Subscriptions } from '@ui/features/stage/subscriptions';
import { getStage } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { LoadingState } from '../common';

export const StageDetails = () => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();

  const { data, isLoading, refetch } = useQuery({
    ...getStage.useQuery({ project: projectName, name: stageName }),
    enabled: !!stageName
  });

  return (
    <Drawer
      open={!!stageName}
      onClose={() => navigate(generatePath(paths.project, { name: projectName }))}
      width={'80%'}
      closable={false}
    >
      {isLoading && <LoadingState />}
      {!isLoading && !data?.stage && <Empty description='Stage not found' />}
      {data?.stage && (
        <>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-center'>
              <HealthStatusIcon
                health={data.stage.status?.currentState?.health}
                style={{ marginRight: '10px', marginTop: '5px' }}
              />
              <Typography.Title level={1} style={{ margin: 0 }}>
                {data.stage.metadata?.name}
              </Typography.Title>
            </div>
            <Typography.Text type='secondary'>{projectName}</Typography.Text>
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8'>
            <Subscriptions
              subscriptions={data.stage.spec?.subscriptions}
              projectName={projectName}
            />
            <AvailableStates stage={data.stage} onSuccess={refetch} />
          </div>
        </>
      )}
    </Drawer>
  );
};
