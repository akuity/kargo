import { Descriptions, Space, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

const WarehouseItem = ({ children }: { children: React.ReactNode }) => (
  <Descriptions bordered size='small' column={1} style={{ width: '50%' }}>
    <Descriptions.Item label='Warehouse'>{children}</Descriptions.Item>
  </Descriptions>
);

const UpstreamStageItem = ({ stage, projectName }: { stage?: string; projectName?: string }) => (
  <Descriptions bordered size='small' key={stage}>
    <Descriptions.Item label='Stage'>
      <Link
        to={generatePath(paths.stage, {
          name: projectName,
          stageName: stage
        })}
      >
        {stage}
      </Link>
    </Descriptions.Item>
  </Descriptions>
);

export const RequestedFreight = (props: { stage?: Stage; projectName?: string }) => {
  const { stage, projectName } = props;
  const subscriptions = stage?.spec?.subscriptions;
  const requestedFreight = stage?.spec?.requestedFreight;
  const uniqueUpstreamStages = new Set<string>();
  for (const freight of requestedFreight || []) {
    for (const stage of freight.sources?.stages || []) {
      uniqueUpstreamStages.add(stage);
    }
  }

  if (!stage) {
    return null;
  }

  return (
    <div>
      <Typography.Title level={3}>Requested Freight</Typography.Title>

      {subscriptions?.warehouse && <WarehouseItem>{subscriptions.warehouse}</WarehouseItem>}

      {(requestedFreight || []).length > 0 && (
        <Space direction='vertical' style={{ width: '50%' }}>
          {requestedFreight?.map((freight) => {
            if (freight.origin?.kind !== 'Warehouse' || !freight.sources?.direct) {
              return <></>;
            }
            return <WarehouseItem key={freight.origin?.name}>{freight.origin?.name}</WarehouseItem>;
          })}
        </Space>
      )}

      {(!!subscriptions?.upstreamStages.length ||
        (Array.from(uniqueUpstreamStages) || []).length > 0) && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Upstream Stages
          </Typography.Title>
          <Space direction='vertical' style={{ width: '50%' }}>
            {subscriptions?.upstreamStages.map((stage) => (
              <UpstreamStageItem stage={stage.name} projectName={projectName} key={stage.name} />
            ))}
            {Array.from(uniqueUpstreamStages).map((stage) => (
              <UpstreamStageItem stage={stage} projectName={projectName} key={stage} />
            ))}
          </Space>
        </>
      )}
    </div>
  );
};
