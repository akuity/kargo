import { Descriptions, Typography } from 'antd';

import { RepoSubscription } from '@ui/gen/api/v1alpha1/generated_pb';
import { urlForImage } from '@ui/utils/url';

type Props = {
  subscriptions?: RepoSubscription[];
  projectName?: string;
};

const DescriptionsLabelStyle: React.CSSProperties = {
  width: '40%'
};

export const RepoSubscriptions = ({ subscriptions }: Props) => {
  if (!subscriptions) {
    return null;
  }

  return (
    <div className='flex flex-col gap-5'>
      {subscriptions.map((subscription) => (
        <>
          {subscription.chart && (
            <Descriptions
              title='Chart'
              bordered
              size='small'
              column={1}
              style={{ width: '40%', minWidth: 500 }}
            >
              <Descriptions.Item label='repo URL' styles={{ label: DescriptionsLabelStyle }}>
                <Typography.Link
                  href={`${subscription.chart?.repoURL}/${subscription?.chart?.name}`}
                  target='_blank'
                  rel='noreferrer'
                >
                  {subscription.chart?.repoURL}
                </Typography.Link>
              </Descriptions.Item>

              {!!subscription?.chart?.discoveryLimit && (
                <Descriptions.Item
                  label='discovery limit'
                  styles={{ label: DescriptionsLabelStyle }}
                >
                  {subscription?.chart?.discoveryLimit}
                </Descriptions.Item>
              )}

              {subscription?.chart?.name && (
                <Descriptions.Item label='name' styles={{ label: DescriptionsLabelStyle }}>
                  {subscription?.chart?.name}
                </Descriptions.Item>
              )}
            </Descriptions>
          )}

          {subscription.git && (
            <Descriptions
              title='Git'
              bordered
              size='small'
              column={1}
              style={{ width: '40%', minWidth: 500 }}
            >
              <Descriptions.Item label='repo URL' styles={{ label: DescriptionsLabelStyle }}>
                <Typography.Link href={subscription.git?.repoURL} target='_blank' rel='noreferrer'>
                  {subscription.git?.repoURL}
                </Typography.Link>
              </Descriptions.Item>

              {!!subscription?.git?.discoveryLimit && (
                <Descriptions.Item
                  label='discovery limit'
                  styles={{ label: DescriptionsLabelStyle }}
                >
                  {subscription?.git?.discoveryLimit}
                </Descriptions.Item>
              )}

              {subscription?.git?.branch && (
                <Descriptions.Item label='branch' styles={{ label: DescriptionsLabelStyle }}>
                  {subscription?.git?.branch}
                </Descriptions.Item>
              )}

              {!!subscription?.git?.semverConstraint && (
                <Descriptions.Item label='constraint' styles={{ label: DescriptionsLabelStyle }}>
                  {subscription?.git?.semverConstraint}
                </Descriptions.Item>
              )}

              {subscription?.git?.commitSelectionStrategy && (
                <Descriptions.Item
                  label='commit selection strategy'
                  styles={{ label: DescriptionsLabelStyle }}
                >
                  {subscription?.git?.commitSelectionStrategy}
                </Descriptions.Item>
              )}
            </Descriptions>
          )}

          {subscription.image && (
            <Descriptions
              title='Image'
              bordered
              size='small'
              column={1}
              style={{ width: '40%', minWidth: 500 }}
            >
              <Descriptions.Item label='repo URL' styles={{ label: DescriptionsLabelStyle }}>
                <Typography.Link
                  href={urlForImage(subscription.image?.repoURL)}
                  target='_blank'
                  rel='noreferrer'
                >
                  {subscription.image?.repoURL}
                </Typography.Link>
              </Descriptions.Item>

              {!!subscription?.image?.discoveryLimit && (
                <Descriptions.Item
                  label='discovery limit'
                  styles={{ label: DescriptionsLabelStyle }}
                >
                  {subscription?.image?.discoveryLimit}
                </Descriptions.Item>
              )}

              {!!subscription?.image?.constraint && (
                <Descriptions.Item label='constraint' styles={{ label: DescriptionsLabelStyle }}>
                  {subscription?.image?.constraint}
                </Descriptions.Item>
              )}

              {subscription?.image?.imageSelectionStrategy && (
                <Descriptions.Item
                  label='image selection strategy'
                  styles={{ label: DescriptionsLabelStyle }}
                >
                  {subscription?.image?.imageSelectionStrategy}
                </Descriptions.Item>
              )}
            </Descriptions>
          )}

          {subscription.subscription && (
            <Descriptions
              title='Other'
              bordered
              size='small'
              column={1}
              style={{ width: '40%', minWidth: 500 }}
            >
              <Descriptions.Item label='Name'>{subscription.subscription.name}</Descriptions.Item>
              <Descriptions.Item label='Type'>
                {subscription.subscription.subscriptionType}
              </Descriptions.Item>
              <Descriptions.Item label='discovery limit'>
                {subscription.subscription.discoveryLimit}
              </Descriptions.Item>
            </Descriptions>
          )}
        </>
      ))}
    </div>
  );
};
