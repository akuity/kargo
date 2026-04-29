import { ColumnType } from 'antd/es/table';

import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import { PhaseCell } from './phase-cell';

export const phaseColumn = (): ColumnType<Stage> => ({
  title: 'Phase',
  width: '10%',
  render: (_, stage) => <PhaseCell stage={stage} />
});
