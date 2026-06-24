import { ColumnType } from 'antd/es/table';

import { Stage } from '@ui/gen/api/v2/models';

import { StageCell } from './stage-cell';

export const stageColumn = (): ColumnType<Stage> => ({
  title: 'Stage',
  width: '20%',
  render: (_, stage) => <StageCell stage={stage} />,
  sorter: (stage1, stage2) =>
    stage1.metadata?.name?.localeCompare(stage2?.metadata?.name || '') || 0
});
