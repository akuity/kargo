import { ColumnType } from 'antd/es/table';
import { formatDistance, isAfter, isBefore } from 'date-fns';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import CustomDatePicker from '@ui/features/common/date-picker';
import {
  Filter,
  useFilterContext
} from '@ui/features/project/pipelines/list/context/filter-context';
import {
  getLastPromotionDate,
  isStageControlFlow
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

export const lastPromotionColumn = (filter: Filter): ColumnType<Stage> => ({
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
  },
  filterDropdown: () => {
    const filters = useFilterContext();

    return (
      <div style={{ padding: 8 }}>
        <CustomDatePicker.RangePicker
          value={filter.lastPromotion}
          onChange={(dates) => {
            filters?.onFilter({
              ...filters.filters,
              // @ts-expect-error expected date
              lastPromotion: dates
            });
          }}
        />
      </div>
    );
  },
  filteredValue: filter?.lastPromotion?.map((d) => d.toString()),
  onFilter: (_, record) => {
    const stageLastPromotion = record?.status?.lastPromotion?.finishedAt;

    if (!filter?.lastPromotion?.filter(Boolean).length) {
      return true;
    }

    const stageLastPromotionDate = timestampDate(stageLastPromotion);

    if (!stageLastPromotionDate) {
      return false;
    }

    return (
      isBefore(stageLastPromotionDate, filter?.lastPromotion[1]) &&
      isAfter(stageLastPromotionDate, filter?.lastPromotion[0])
    );
  }
});
