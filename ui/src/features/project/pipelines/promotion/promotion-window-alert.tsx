import { Alert, Typography } from 'antd';
import { format, formatDistance } from 'date-fns';

import type { Stage } from '@ui/gen/api/v2/models';

import {
  isPromotionWindowClosed,
  promotionWindowClosedReason,
  promotionWindowNextOpen
} from './promotion-window';

export const PromotionWindowAlert = ({
  stage,
  className
}: {
  stage: Stage;
  className?: string;
}) => {
  if (!isPromotionWindowClosed(stage)) {
    return null;
  }

  const nextOpen = promotionWindowNextOpen(stage);

  const reopens = nextOpen
    ? `Reopens ${formatDistance(nextOpen, new Date(), {
        addSuffix: true
      })}, on ${format(nextOpen, 'MMM do yyyy HH:mm:ss')}.`
    : 'No reopening time is known, so this Stage stays frozen until the schedule changes.';

  // an Alert without a description keeps antd's compact padding
  return (
    <Alert
      className={className}
      type='warning'
      message={
        <span className='text-sm'>
          <Typography.Text strong className='mr-2'>
            Promotions frozen
          </Typography.Text>
          <Typography.Text type='secondary'>
            {promotionWindowClosedReason(stage)} {reopens}
          </Typography.Text>
        </span>
      }
    />
  );
};
