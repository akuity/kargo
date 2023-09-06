import { useQuery } from '@tanstack/react-query';
import { Divider, Drawer, Empty, Tabs, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Subscriptions } from '@ui/features/stage/subscriptions';
import { getStage } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { AvailableFreight } from './available-freight';
import { ManifestPreview } from './manifest-preview';
import { StageActions } from './stage-actions';

export const StageDetails = () => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();

  const { data, isLoading, refetch } = useQuery({
    ...getStage.useQuery({ project: projectName, name: stageName }),
    enabled: !!stageName
  });

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {isLoading && <LoadingState />}
      {!isLoading && !data?.stage && <Empty description='Stage not found' />}
      {data?.stage && (
        <>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <HealthStatusIcon
                health={data.stage.status?.currentFreight?.health}
                style={{ marginRight: '10px', marginTop: '10px' }}
              />
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {data.stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
              </div>
            </div>
            <StageActions stage={data.stage} />
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8'>
            <Subscriptions
              subscriptions={data.stage.spec?.subscriptions}
              projectName={projectName}
            />
            <Tabs
              defaultActiveKey='1'
              items={[
                {
                  key: '1',
                  label: 'Available Freight',
                  children: <AvailableFreight stage={data.stage} onSuccess={refetch} />
                },
                {
                  key: '2',
                  label: 'Manifest Preview',
                  children: <ManifestPreview stage={data.stage} />
                }
              ]}
            />
          </div>
        </>
      )}
    </Drawer>
  );
};
