import { useQuery } from '@connectrpc/connect-query';
import { faCircleNotch, faCodePullRequest } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Spin, Tag } from 'antd';
import Link from 'antd/es/typography/Link';
import { useMemo } from 'react';

import { getPromotionOutputsByStepAlias } from '@ui/features/stage/utils/promotion';
import { getPromotion } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Promotion, Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { getPromotionStepAlias } from '@ui/plugins/atoms/plugin-helper';

import { getCurrentPromotion } from './stage-meta-utils';

type PullRequestLinkProps = {
  stage: Stage;
};

export const PullRequestLink = (props: PullRequestLinkProps) => {
  const currentPromotion = getCurrentPromotion(props.stage);

  const getPromotionQuery = useQuery(
    getPromotion,
    { project: props.stage?.metadata?.namespace, name: currentPromotion },
    {
      enabled:
        !!currentPromotion &&
        !!props.stage?.spec?.promotionTemplate?.spec?.steps?.find(
          (step) => step?.uses === 'git-open-pr'
        )
    }
  );

  const promotion = getPromotionQuery.data?.result?.value as Promotion;

  const outputsByStepAlias: Record<string, object> = useMemo(
    () => getPromotionOutputsByStepAlias(promotion),
    [promotion]
  );

  const indexOfPullRequest = promotion?.spec?.steps?.findIndex(
    (step) => step?.uses === 'git-open-pr'
  );

  if (getPromotionQuery.isFetching) {
    return <Spin size='small' />;
  }

  // type safe
  if (!promotion || !promotion.spec || !promotion.spec.steps) {
    return null;
  }

  if (!indexOfPullRequest || indexOfPullRequest < 0) {
    return null;
  }

  const hasPullRequestStepSucceeded =
    promotion?.status?.stepExecutionMetadata[indexOfPullRequest]?.status === 'Succeeded';

  if (!hasPullRequestStepSucceeded) {
    return null;
  }

  const aliasOfPullRequestStep = getPromotionStepAlias(
    promotion.spec.steps[indexOfPullRequest],
    indexOfPullRequest
  );

  const outputOfPullRequestStep = outputsByStepAlias?.[aliasOfPullRequestStep];

  const pullRequestLink = (outputOfPullRequestStep as { pr?: { url?: string } }).pr?.url;

  if (!pullRequestLink) {
    return null;
  }

  return (
    <Link href={pullRequestLink} target='_blank'>
      <Tag color='orange'>
        <span className='text-[10px]'>
          Waiting for Approval <FontAwesomeIcon className='ml-1' icon={faCodePullRequest} />
          <FontAwesomeIcon icon={faCircleNotch} spin className='ml-1' />
        </span>
      </Tag>
    </Link>
  );
};
