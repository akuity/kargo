import { IconProp } from '@fortawesome/fontawesome-svg-core';
import {
  faAnchor,
  faCodeBranch,
  faCube,
  faPuzzlePiece,
  faQuestion
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { RepoSubscription } from '@ui/extend/types';

import { artifactURL, humanComprehendableArtifact } from '../freight/artifact-parts-utils';

import styles from './node-size-source-of-truth.module.less';
import warehouseStyles from './warehouse-node.module.less';
import { subscriptionRepoURL } from './warehouse-state';

const subscriptionKind = (subscription: RepoSubscription): { icon: IconProp; label: string } => {
  if (subscription?.chart) {
    return { icon: faAnchor, label: 'Chart' };
  }
  if (subscription?.git) {
    return { icon: faCodeBranch, label: 'Git' };
  }
  if (subscription?.image) {
    return { icon: faCube, label: 'Image' };
  }
  if (subscription?.subscription) {
    return { icon: faPuzzlePiece, label: subscription.subscription.subscriptionType || 'Custom' };
  }
  return { icon: faQuestion, label: 'Unknown' };
};

export const SubscriptionNode = (props: {
  subscription: RepoSubscription;
  // this is the subscription the Warehouse's discovery failed on
  failing?: boolean;
}) => {
  const { title, repoURL } = useMemo(() => {
    const repoURL = subscriptionRepoURL(props.subscription);
    const title = props.subscription.name || humanComprehendableArtifact({ repoURL }) || repoURL;

    return { title, repoURL };
  }, [props.subscription]);

  const kind = subscriptionKind(props.subscription);

  return (
    <Card
      size='small'
      style={{ borderRadius: 10 }}
      styles={{ body: { padding: '9px 12px' } }}
      className={classNames(warehouseStyles.subscriptionNode, styles['subscription-node-size'], {
        [warehouseStyles.failing]: props.failing
      })}
    >
      <div className={warehouseStyles.subscriptionRow}>
        <FontAwesomeIcon
          icon={kind.icon}
          className={warehouseStyles.subscriptionIcon}
          aria-hidden
        />
        <span className={warehouseStyles.subscriptionName} title={title}>
          {title}
        </span>
        {props.failing ? (
          <span className={warehouseStyles.subscriptionFailingLabel}>Failing</span>
        ) : (
          <span className={warehouseStyles.subscriptionKind} title={kind.label}>
            {kind.label}
          </span>
        )}
      </div>

      {repoURL ? (
        <a
          href={artifactURL(repoURL)}
          target='_blank'
          rel='noreferrer'
          className={warehouseStyles.subscriptionURL}
          title={repoURL}
        >
          {repoURL} ↗
        </a>
      ) : (
        <div
          className={classNames(
            warehouseStyles.subscriptionURL,
            warehouseStyles.subscriptionURLPlain
          )}
        >
          Custom subscription
        </div>
      )}
    </Card>
  );
};
