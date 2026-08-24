import { Descriptions, Typography } from 'antd';

import { RepoSubscription } from '@ui/extend/types';
import { SubscriptionNameTag } from '@ui/features/common/subscription-name-tag';
import { urlForImage } from '@ui/utils/url';

type Props = {
  subscriptions?: RepoSubscription[];
  projectName?: string;
};

const DescriptionsLabelStyle: React.CSSProperties = {
  width: '40%'
};

// subscriptionTitle pairs the kind of a subscription with its optional name,
// which is what identifies one subscription among several of the same kind.
const subscriptionTitle = (kind: string, name?: string) => (
  <span className='flex items-center gap-2'>
    {kind}
    <SubscriptionNameTag name={name} className='mr-0' />
  </span>
);

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
              title={subscriptionTitle('Chart', subscription.name)}
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
              title={subscriptionTitle('Git', subscription.name)}
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
              title={subscriptionTitle('Image', subscription.name)}
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
              title={subscriptionTitle('Other', subscription.name)}
              bordered
              size='small'
              column={1}
              style={{ width: '40%', minWidth: 500 }}
            >
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
