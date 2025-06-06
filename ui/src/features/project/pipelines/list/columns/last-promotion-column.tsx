import { ColumnType } from 'antd/es/table';
import { formatDistance } from 'date-fns';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  getLastPromotionDate,
  isStageControlFlow
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

export const lastPromotionColumn = (): ColumnType<Stage> => ({
  title: 'Last Promotion',
  width: '15%',
  render: (_, stage) => {
    if (isStageControlFlow(stage)) {
      return '-';
    }

    const lastPromotion = getLastPromotionDate(stage);

    if (!lastPromotion) {
      return '-';
    }

    const date = timestampDate(lastPromotion) as Date;

    return (
      <Link
        to={generatePath(paths.promotion, {
          name: stage?.metadata?.namespace,
          promotionId: stage?.status?.lastPromotion?.name
        })}
      >
        {formatDistance(date, new Date(), { addSuffix: true })}
      </Link>
    );
  }
});
