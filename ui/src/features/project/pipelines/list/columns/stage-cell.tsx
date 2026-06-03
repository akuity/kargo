import { Badge, Space } from 'antd';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  isStageControlFlow,
  useStageHeaderStyle
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const StageCell = ({ stage }: { stage: Stage }) => {
  const stageHeader = useStageHeaderStyle(stage);
  const background = stageHeader?.backgroundColor;

  return (
    <Space size={6} wrap>
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
    </Space>
  );
};
