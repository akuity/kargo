import { useQuery } from '@connectrpc/connect-query';
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

import { getPromotionOutputsByStepAlias } from '@ui/features/stage/utils/promotion';
import { getPromotion } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Promotion, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';

import { getCurrentPromotion } from './stage-meta-utils';

type PullRequestLinkProps = {
  stage: Stage;
  className?: string;
};

export const PullRequestLink = (props: PullRequestLinkProps) => {
  const currentPromotion = getCurrentPromotion(props.stage);

  const getPromotionQuery = useQuery(
    getPromotion,
    { project: props.stage?.metadata?.namespace, name: currentPromotion },
    {
      enabled: !!currentPromotion
    }
  );

  const promotion = getPromotionQuery.data?.result?.value as Promotion;

  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(promotion),
    [promotion]
  );

  const indexOfPullRequest = promotion?.spec?.steps?.findIndex(
    (step: { uses?: string }) => step?.uses === 'git-open-pr' || step?.uses === 'git-wait-for-pr'
  );

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
  const stepType = step?.uses;
  const stepMetadata = promotion?.status?.stepExecutionMetadata?.[indexOfPullRequest];
  const stepStatus = stepMetadata?.status;

  const aliasOfPullRequestStep = getPromotionStepAlias(step, indexOfPullRequest);

  const outputOfPullRequestStep = outputsByStepAlias?.[aliasOfPullRequestStep];

  const pullRequestLink = (outputOfPullRequestStep as { pr?: { url?: string } })?.pr?.url;

  if (!pullRequestLink) {
    return null;
  }

  // For git-open-pr: only show when succeeded
  // For git-wait-for-pr: show when running (has PR URL) or succeeded
  const isGitWaitForPr = stepType === 'git-wait-for-pr';
  const isGitOpenPr = stepType === 'git-open-pr';
  const hasPullRequestStepSucceeded = stepStatus === 'Succeeded';
  const hasPullRequestStepRunning = stepStatus === 'Running';

  const isStatusAcceptable =
    (isGitOpenPr && hasPullRequestStepSucceeded) ||
    (isGitWaitForPr && (hasPullRequestStepSucceeded || hasPullRequestStepRunning));

  if (!isStatusAcceptable) {
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
