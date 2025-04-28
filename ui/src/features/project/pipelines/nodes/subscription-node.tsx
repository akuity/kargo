import { IconProp } from '@fortawesome/fontawesome-svg-core';
import { faDocker, faGitAlt } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faExternalLink, faQuestion } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Flex, Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import { useMemo } from 'react';

import { RepoSubscription } from '@ui/gen/api/v1alpha1/generated_pb';

import {
  artifactBase,
  artifactURL,
  humanComprehendableArtifact
} from '../freight/artifact-parts-utils';

import styles from './node-size-source-of-truth.module.less';

export const SubscriptionNode = (props: { subscription: RepoSubscription }) => {
  const { title, base, link } = useMemo(() => {
    const repoURL =
      props.subscription?.git?.repoURL ||
      props.subscription?.chart?.repoURL ||
      props.subscription?.image?.repoURL ||
      '';
    const title = humanComprehendableArtifact(repoURL);
    const base = artifactBase(repoURL);
    const link = artifactURL(repoURL);

    return { title, repoURL, base, link };
  }, [props.subscription]);

  let icon: IconProp = faQuestion;

  if (props.subscription?.chart) {
    icon = faAnchor;
  } else if (props.subscription?.git) {
    icon = faGitAlt;
  } else if (props.subscription?.image) {
    icon = faDocker;
  }

  return (
    <Card
      size='small'
      className={styles['subscription-node-size']}
      title={
        <Flex align='center' gap={16}>
          <FontAwesomeIcon icon={icon} />
          <span className='text-xs'>{title}</span>
        </Flex>
      }
      variant='borderless'
    >
      <Link href={link} target='_blank'>
        <Tag className='text-[9px] text-wrap' color='blue' bordered={false}>
          {base}

          <FontAwesomeIcon icon={faExternalLink} className='ml-1' />
        </Tag>
      </Link>
    </Card>
  );
};
