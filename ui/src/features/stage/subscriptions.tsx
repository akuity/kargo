import { Descriptions, Space, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Subscriptions as SubscriptionsType } from '@ui/gen/v1alpha1/generated_pb';

export const Subscriptions = (props: {
  subscriptions?: SubscriptionsType;
  projectName?: string;
}) => {
  const { subscriptions, projectName } = props;

  if (!subscriptions) {
    return null;
  }

  return (
    <div>
      <Typography.Title level={3}>Subscriptions</Typography.Title>

      {subscriptions.warehouse && (
        <Descriptions bordered size='small' column={1} style={{ width: '50%' }}>
          <Descriptions.Item label='Warehouse'>{subscriptions.warehouse}</Descriptions.Item>
        </Descriptions>
      )}

      {!!subscriptions.upstreamStages.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Upstream Stages
          </Typography.Title>
          <Space direction='vertical' style={{ width: '50%' }}>
            {subscriptions?.upstreamStages.map((stage) => (
              <Descriptions bordered size='small' key={stage.name} column={1}>
                <Descriptions.Item label='Stage'>
                  <Link
                    to={generatePath(paths.stage, {
                      name: projectName,
                      stageName: stage.name
                    })}
                  >
                    {stage.name}
                  </Link>
                </Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}
    </div>
  );
};
