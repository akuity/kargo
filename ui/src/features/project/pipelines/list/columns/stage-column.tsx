import { Badge } from 'antd';
import { ColumnType } from 'antd/es/table';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  isStageControlFlow,
  useStageHeaderStyle
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const stageColumn = (): ColumnType<Stage> => ({
  title: 'Stage',
  width: '20%',
  render: (_, stage) => {
    const stageHeader = useStageHeaderStyle(stage);

    const background = stageHeader?.backgroundColor;

    return (
      <Link
        to={generatePath(paths.stage, {
          name: stage?.metadata?.namespace,
          stageName: stage?.metadata?.name
        })}
      >
        {!!background && <Badge color={background} className='mr-2' />}
        {stage?.metadata?.name}
        {isStageControlFlow(stage) ? <span className='text-xs ml-1'>(Control Flow)</span> : ''}
      </Link>
    );
  },
  sorter: (stage1, stage2) =>
    stage1.metadata?.name?.localeCompare(stage2?.metadata?.name || '') || 0
});
