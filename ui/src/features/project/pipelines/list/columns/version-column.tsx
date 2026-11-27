import { ColumnType } from 'antd/es/table';

import { StageFreight } from '@ui/features/project/pipelines/nodes/stage-freight';
import { isStageControlFlow } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

export const versionColumn = (): ColumnType<Stage> => ({
  title: 'Version',
  // The Stage's current freight, as its node in the graph shows it and as wide
  // as it is there, less the caption the column header already gives. A
  // control-flow Stage never holds freight, so -- as its node does -- it shows
  // none.
  render: (_, stage) =>
    isStageControlFlow(stage) ? (
      '-'
    ) : (
      <div className='w-[280px]'>
        <StageFreight stage={stage} hideLabel />
      </div>
    )
});
