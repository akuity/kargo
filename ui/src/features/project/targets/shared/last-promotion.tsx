import { Flex, Tooltip, Typography } from 'antd';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { rolloutPath } from '@ui/features/stage/utils/promotion-rollout';
import { TargetStatusEntry } from '@ui/gen/api/v2/models';

import { promotionWhen } from '../model/target-details-utils';

/**
 * LastPromotion is a Stage's newest Promotion to a Target in one line: its
 * phase, a link to it, the rollout that made it, and when. It is the fact a
 * Target's state is read from, so wherever a state is shown this can say why.
 */
export const LastPromotion = ({
  project,
  entry,
  freightLabel
}: {
  project: string;
  entry: TargetStatusEntry;
  freightLabel: (name: string) => string;
}) => {
  const promotion = entry.lastPromotion;
  const stageName = entry.stage || '';
  const rolloutLink = (number: number) => (
    <Link to={rolloutPath(project, stageName, number)}>rollout #{number}</Link>
  );

  return (
    <Flex vertical gap={2} align='flex-end'>
      {promotion?.name ? (
        <Flex gap={6} align='center' wrap justify='flex-end'>
          <PromotionStatusIcon status={{ phase: promotion.phase }} />
          <Link to={generatePath(paths.promotion, { name: project, promotionId: promotion.name })}>
            {promotion.phase || 'Pending'}
          </Link>
          {promotion.freight && (
            <span className='font-mono'>{freightLabel(promotion.freight)}</span>
          )}
          {promotion.rollout ? <span>· {rolloutLink(promotion.rollout)}</span> : null}
          <Tooltip title={promotionWhen(promotion).date?.toLocaleString()}>
            <Typography.Text type='secondary' className='text-xs'>
              {promotionWhen(promotion).text}
            </Typography.Text>
          </Tooltip>
        </Flex>
      ) : (
        <Typography.Text type='secondary' className='text-xs'>
          none
        </Typography.Text>
      )}
      {/* A state a rollout explains rather than a Promotion: one still to
          reach the Target, or one that ended without doing so. */}
      {entry.rollout && entry.state === 'NotAttempted' ? (
        <Typography.Text type='secondary' className='text-xs'>
          not attempted by {rolloutLink(entry.rollout)}
        </Typography.Text>
      ) : null}
      {entry.rollout && entry.state === 'Pending' ? (
        <Typography.Text type='secondary' className='text-xs'>
          waiting for {rolloutLink(entry.rollout)}
        </Typography.Text>
      ) : null}
    </Flex>
  );
};
