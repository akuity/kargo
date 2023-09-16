import { Descriptions, Space, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { Subscriptions as SubscriptionsType } from '@ui/gen/v1alpha1/types_pb';
import { urlWithProtocol } from '@ui/utils/url';

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

      {!!subscriptions.repos?.git.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Git Repositories
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.git.map((gitRepo) => (
              <Descriptions bordered size='small' key={gitRepo.repoUrl} column={1}>
                <Descriptions.Item label='URL'>
                  <Typography.Link
                    href={urlWithProtocol(gitRepo.repoUrl)}
                    target='_blank'
                    rel='noreferrer'
                  >
                    {gitRepo.repoUrl}
                  </Typography.Link>
                </Descriptions.Item>
                <Descriptions.Item label='Branch'>{gitRepo.branch}</Descriptions.Item>
              </Descriptions>
            ))}
          </Space>
        </>
      )}

      {!!subscriptions.repos?.images.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Container images
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.images.map((image) => (
              <Descriptions bordered size='small' key={image.repoUrl}>
                <Descriptions.Item label='URL'>
                  <Typography.Link
                    href={urlWithProtocol(image.repoUrl)}
                    target='_blank'
                    rel='noreferrer'
                  >
                    {image.repoUrl}
                  </Typography.Link>
                </Descriptions.Item>
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

      {!!subscriptions.repos?.charts.length && (
        <>
          <Typography.Title level={5} style={{ marginTop: '.8em' }}>
            Helm charts
          </Typography.Title>
          <Space direction='vertical' style={{ width: '100%' }}>
            {subscriptions?.repos.charts.map((chart) => (
              <Descriptions bordered size='small' key={chart.registryUrl}>
                <Descriptions.Item label='Registry URL'>
                  <Typography.Link
                    href={urlWithProtocol(chart.registryUrl)}
                    target='_blank'
                    rel='noreferrer'
                  >
                    {chart.registryUrl}
                  </Typography.Link>
                </Descriptions.Item>
                {chart.name && <Descriptions.Item label='Name'>{chart.name}</Descriptions.Item>}
                {chart.semverConstraint && (
                  <Descriptions.Item label='Semver Constraint'>
                    {chart.semverConstraint}
                  </Descriptions.Item>
                )}
              </Descriptions>
            ))}
          </Space>
        </>
      )}
    </div>
  );
};
