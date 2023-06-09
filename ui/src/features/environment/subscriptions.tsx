import { paths } from '@config/paths';
import { Subscriptions as SubscriptionsType } from '@gen/v1alpha1/generated_pb';
import { Descriptions, Space, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

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

      {subscriptions.upstreamEnvs.length > 0 && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Upstream Environments
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.upstreamEnvs.map((env) => (
              <Descriptions bordered size='small' key={env.name} column={1}>
                <Descriptions.Item label='Environment'>
                  <Link
                    to={generatePath(paths.environment, {
                      name: projectName,
                      environmentName: env.name
                    })}
                  >
                    {env.name}
                  </Link>
                </Descriptions.Item>
                <Descriptions.Item label='Project'>{env.namespace}</Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}

      {subscriptions.repos?.git && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Git Repositories
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.git.map((gitRepo) => (
              <Descriptions bordered size='small' key={gitRepo.repoURL} column={1}>
                <Descriptions.Item label='URL'>{gitRepo.repoURL}</Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}

      {subscriptions.repos?.images && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Images
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.images.map((image) => (
              <Descriptions bordered size='small' key={image.repoURL}>
                <Descriptions.Item label='URL'>{image.repoURL}</Descriptions.Item>
                <Descriptions.Item label='Semver Constraint'>
                  {image.semverConstraint}
                </Descriptions.Item>
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
