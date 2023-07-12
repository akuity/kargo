import { Divider, Typography } from 'antd';
import { useParams } from 'react-router-dom';

import { HealthStatusIcon } from '@ui/features/common/health-status-icon/health-status-icon';
import { AvailableStates } from '@ui/features/stage/available-states';
import { Subscriptions } from '@ui/features/stage/subscriptions';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

export const StageDetails = (props: { stage: Stage; refetch: () => void }) => {
  const { stage, refetch } = props;
  const { name: projectName } = useParams();

  return (
    <>
      <div className='flex items-center justify-between'>
        <div className='flex gap-1 items-center'>
          <HealthStatusIcon
            health={stage?.status?.currentState?.health}
            style={{ marginRight: '10px', marginTop: '5px' }}
          />
          <Typography.Title level={1} style={{ margin: 0 }}>
            {stage?.metadata?.name}
          </Typography.Title>
        </div>
        <Typography.Text type='secondary'>{projectName}</Typography.Text>
      </div>
      <Divider style={{ marginTop: '1em' }} />

      <div className='flex flex-col gap-8'>
        <Subscriptions subscriptions={stage?.spec?.subscriptions} projectName={projectName} />
        <AvailableStates stage={stage} onSuccess={refetch} />
      </div>
    </>
  );
};
