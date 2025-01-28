import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faQuestion } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';

import { RepoSubscription } from '@ui/gen/v1alpha1/generated_pb';

import styles from './custom-node.module.less';

type SubscriptionNodeProps = {
  subscription: RepoSubscription;
};

export const SubscriptionNode = (props: SubscriptionNodeProps) => {
  let icon: IconProp = faQuestion;

  if (props.subscription?.chart) {
    icon = faAnchor;
  } else if (props.subscription?.git) {
    icon = faGitAlt;
  } else if (props.subscription?.image) {
    icon = faDocker;
  }

  const url =
    props.subscription?.git?.repoURL ||
    props.subscription?.image?.repoURL ||
    props.subscription?.chart?.repoURL;

  return (
    <div className={classNames(styles.repoSubscriptionNode)}>
      <div className={classNames(styles.header)}>
        <h3>Subscription</h3>

        <FontAwesomeIcon className='ml-auto text-base' icon={icon} />
      </div>

      <div className={classNames(styles.body)}>
        <Tooltip title={url}>
          <span className='block w-36 overflow-hidden text-ellipsis whitespace-nowrap'>{url}</span>
        </Tooltip>
      </div>
    </div>
  );
};
