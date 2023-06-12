import { AvailableStates } from '@features/environment/available-states';
import { Subscriptions } from '@features/environment/subscriptions';
import { HealthStatusIcon } from '@features/ui/health-status-icon/health-status-icon';
import { Environment } from '@gen/v1alpha1/generated_pb';
import { Divider, Typography } from 'antd';
import { useParams } from 'react-router-dom';

export const EnvironmentDetails = (props: { environment: Environment; refetch: () => void }) => {
  const { environment, refetch } = props;
  const { name: projectName } = useParams();

  return (
    <>
      <div className='flex items-center justify-between'>
        <div className='flex gap-1 items-center'>
          <HealthStatusIcon
            health={environment?.status?.currentState?.health}
            style={{ marginRight: '10px', marginTop: '5px' }}
          />
          <Typography.Title level={1} style={{ margin: 0 }}>
            {environment?.metadata?.name}
          </Typography.Title>
        </div>
        <Typography.Text type='secondary'>{projectName}</Typography.Text>
      </div>
      <Divider style={{ marginTop: '1em' }} />

      <div className='flex flex-col gap-8'>
        <Subscriptions subscriptions={environment?.spec?.subscriptions} projectName={projectName} />
        <AvailableStates environment={environment} onSuccess={refetch} />
      </div>
    </>
  );
};
