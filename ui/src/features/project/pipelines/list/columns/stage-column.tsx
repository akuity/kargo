import { ColumnType } from 'antd/es/table';

import { Filter } from '@ui/features/project/pipelines/list/context/filter-context';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { StageCell } from './stage-cell';
import { StageFilterDropdown } from './stage-filter-dropdown';

export const stageColumn = (filter: Filter): ColumnType<Stage> => ({
  title: 'Stage',
  width: '20%',
  render: (_, stage) => <StageCell stage={stage} />,
  filterDropdown: (props) => <StageFilterDropdown {...props} />,
  onFilter: (value, record) => {
    return !!record?.metadata?.name?.toLowerCase?.()?.includes((value as string).toLowerCase());
  },
  filteredValue: filter?.stage ? [filter.stage] : null,
  filtered: !!filter?.stage,
  sorter: (stage1, stage2) =>
    stage1.metadata?.name?.localeCompare(stage2?.metadata?.name || '') || 0
});
