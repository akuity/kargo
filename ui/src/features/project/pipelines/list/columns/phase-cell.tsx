import { Flex } from 'antd';
import { Link } from 'react-router-dom';

import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import { getCurrentPromotionRef } from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import { Stage } from '@ui/gen/api/v2/models';

export const PhaseCell = ({ stage }: { stage: Stage }) => {
  const { controllerName, isControllerDead } = useStageControllerStatus(stage);
  const stagePhase = getStagePhase(stage, isControllerDead);

  if (getCurrentFreight(stage).length === 0) {
    return <>-</>;
  }

  const Comp = (
    <Flex align='center' gap={4}>
      {stagePhase}{' '}
      <StageConditionIcon
        conditions={stage?.status?.conditions || []}
        noTooltip
        className='text-[10px]'
        isControllerDead={isControllerDead}
        controllerName={controllerName}
      />
    </Flex>
  );

  const currentPromotionPath = getCurrentPromotionRef(stage)?.path;

  if (stagePhase === 'Promoting' && currentPromotionPath) {
    return <Link to={currentPromotionPath}>{Comp}</Link>;
  }

  return Comp;
};
