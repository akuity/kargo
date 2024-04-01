import { Divider, Drawer, Tabs, Typography } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { Subscriptions } from '@ui/features/stage/subscriptions';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

import { ManifestPreview } from './manifest-preview';
import { Promotions } from './promotions';
import { StageActions } from './stage-actions';
import { Verifications } from './verifications';

export const StageDetails = ({ stage }: { stage: Stage }) => {
  const { name: projectName, stageName } = useParams();
  const navigate = useNavigate();

  const onClose = () => navigate(generatePath(paths.project, { name: projectName }));

  return (
    <Drawer open={!!stageName} onClose={onClose} width={'80%'} closable={false}>
      {stage && (
        <div className='flex flex-col h-full'>
          <div className='flex items-center justify-between'>
            <div className='flex gap-1 items-start'>
              <HealthStatusIcon
                health={stage.status?.health}
                style={{ marginRight: '10px', marginTop: '10px' }}
              />
              <div>
                <Typography.Title level={1} style={{ margin: 0 }}>
                  {stage.metadata?.name}
                </Typography.Title>
                <Typography.Text type='secondary'>{projectName}</Typography.Text>
              </div>
            </div>
            <StageActions stage={stage} />
          </div>
          <Divider style={{ marginTop: '1em' }} />

          <div className='flex flex-col gap-8 flex-1'>
            <Subscriptions subscriptions={stage.spec?.subscriptions} projectName={projectName} />
            <Tabs
              className='flex-1'
              defaultActiveKey='1'
              style={{ minHeight: '500px' }}
              items={[
                {
                  key: '1',
                  label: 'Promotions',
                  children: <Promotions />
                },
                {
                  key: '2',
                  label: 'Verifications',
                  children: <Verifications stage={stage} />
                },
                {
                  key: '3',
                  label: 'Live Manifest',
                  className: 'h-full pb-2',
                  children: <ManifestPreview stage={stage} />
                }
              ]}
            />
          </div>
        </div>
      )}
    </Drawer>
  );
};
