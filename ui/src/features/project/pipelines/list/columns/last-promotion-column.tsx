import { ColumnType } from 'antd/es/table';
import { formatDistance } from 'date-fns';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  getLastPromotionDate,
  isStageControlFlow
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

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

    const date = lastPromotion;

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
  },
  sorter: (stage1, stage2) => {
    if (isStageControlFlow(stage1)) {
      return -1;
    }

    if (isStageControlFlow(stage2)) {
      return 1;
    }

    const stage1LastPromotionDate = getLastPromotionDate(stage1);
    const stage2LastPromotionDate = getLastPromotionDate(stage2);

    if (!stage1LastPromotionDate) {
      return 1;
    }
    if (!stage2LastPromotionDate) {
      return -1;
    }

    return stage2LastPromotionDate > stage1LastPromotionDate ? 1 : -1;
  }
});
