import { Descriptions, Space, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Subscriptions as SubscriptionsType } from '@ui/gen/v1alpha1/types_pb';

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

      {!!subscriptions.upstreamStages.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Upstream Stages
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
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

      {!!subscriptions.repos?.git.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Git Repositories
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.git.map((gitRepo) => (
              <Descriptions bordered size='small' key={gitRepo.repoUrl} column={1}>
                <Descriptions.Item label='URL'>{gitRepo.repoUrl}</Descriptions.Item>
                <Descriptions.Item label='Branch'>{gitRepo.branch}</Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}

      {!!subscriptions.repos?.images.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Images
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.images.map((image) => (
              <Descriptions bordered size='small' key={image.repoUrl}>
                <Descriptions.Item label='URL'>{image.repoUrl}</Descriptions.Item>
                {image.semverConstraint && (
                  <Descriptions.Item label='Semver Constraint'>
                    {image.semverConstraint}
                  </Descriptions.Item>
                )}
                <Descriptions.Item label='Update Strategy'>
                  {image.updateStrategy}
                </Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}
    </div>
  );
};
