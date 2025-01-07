import { Descriptions, Typography } from 'antd';

import { RepoSubscription } from '@ui/gen/v1alpha1/generated_pb';
import { urlForImage } from '@ui/utils/url';

type Props = {
  subscriptions?: RepoSubscription[];
  projectName?: string;
};

export const RepoSubscriptions = ({ subscriptions }: Props) => {
  if (!subscriptions) {
    return null;
  }

  return (
    <div>
      <Descriptions bordered size='small' column={1} style={{ width: '50%', minWidth: 500 }}>
        {subscriptions.map((subscription) => (
          <>
            {subscription.chart && (
              <Descriptions.Item label='Chart'>
                <Typography.Link
                  href={subscription.chart?.repoURL}
                  target='_blank'
                  rel='noreferrer'
                >
                  {subscription.chart?.repoURL}
                </Typography.Link>
              </Descriptions.Item>
            )}
            {subscription.git && (
              <Descriptions.Item label='Git'>
                <Typography.Link href={subscription.git?.repoURL} target='_blank' rel='noreferrer'>
                  {subscription.git?.repoURL}
                </Typography.Link>
              </Descriptions.Item>
            )}
            {subscription.image && (
              <Descriptions.Item label='Image'>
                <Typography.Link
                  href={urlForImage(subscription.image?.repoURL)}
                  target='_blank'
                  rel='noreferrer'
                >
                  {subscription.image?.repoURL}
                </Typography.Link>
              </Descriptions.Item>
            )}
          </>
        ))}
      </Descriptions>
    </div>
  );
};
