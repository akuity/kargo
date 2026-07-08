import {
  faCircleNotch,
  faCodePullRequest,
  faExternalLink
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Spin, Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import classNames from 'classnames';
import { useMemo } from 'react';

import { isPromotionStepStatusTerminal } from '@ui/features/common/promotion-status/utils';
import { getPromotionOutputsByStepAlias } from '@ui/features/stage/utils/promotion';
import { useGetPromotion } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';
import { getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';

import { getCurrentPromotion } from './stage-meta-utils';

type PullRequestLinkProps = {
  stage: Stage;
  className?: string;
};

export const PullRequestLink = (props: PullRequestLinkProps) => {
  const currentPromotion = getCurrentPromotion(props.stage);

  const getPromotionQuery = useGetPromotion(
    props.stage?.metadata?.namespace || '',
    currentPromotion || '',
    {
      query: { enabled: !!currentPromotion }
    }
  );

  const promotion = getPromotionQuery.data?.data;

  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(promotion),
    [promotion]
  );

  // "Waiting for Approval" tracks the git-wait-for-pr step, which is the only
  // step that actually waits on a pull request. git-open-pr succeeds as soon as
  // the PR is opened and never represents a waiting state.
  const indexOfPullRequest =
    promotion?.spec?.steps?.findIndex(
      (step: { uses?: string }) => step?.uses === 'git-wait-for-pr'
    ) ?? -1;

  if (getPromotionQuery.isFetching) {
    return <Spin size='small' />;
  }

  // type safe
  if (!promotion || !promotion.spec || !promotion.spec.steps) {
    return null;
  }

  if (indexOfPullRequest === undefined || indexOfPullRequest < 0) {
    return null;
  }

  const step = promotion.spec.steps[indexOfPullRequest];
  const stepMetadata = promotion?.status?.stepExecutionMetadata?.[indexOfPullRequest];
  const stepStatus = stepMetadata?.status;

  const aliasOfPullRequestStep = getPromotionStepAlias(step, indexOfPullRequest);

  const outputOfPullRequestStep = outputsByStepAlias?.[aliasOfPullRequestStep];

  const pullRequestLink = (outputOfPullRequestStep as { pr?: { url?: string } })?.pr?.url;

  if (!pullRequestLink) {
    return null;
  }

  // Only surface "Waiting for Approval" while the pull request step has not yet
  // terminated. Once it reaches a terminal status (e.g. the PR is merged or the
  // step fails), the Promotion is no longer waiting on approval.
  if (isPromotionStepStatusTerminal(stepStatus)) {
    return null;
  }

  return (
    <Link href={pullRequestLink} target='_blank' className={classNames(props.className)}>
      <Tag color='green' bordered={false}>
        <span className='text-[8px]'>
          Waiting for Approval <FontAwesomeIcon className='ml-1' icon={faCodePullRequest} />
          <FontAwesomeIcon icon={faCircleNotch} spin className='ml-1' />
          <FontAwesomeIcon icon={faExternalLink} className='ml-1' />
        </span>
      </Tag>
    </Link>
  );
};
