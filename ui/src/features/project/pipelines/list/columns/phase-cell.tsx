import { Flex } from 'antd';
import { Link } from 'react-router-dom';

import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import { useCurrentPromotionRef } from '@ui/features/stage/use-promotion-rollouts';
import { Stage } from '@ui/gen/api/v2/models';

export const PhaseCell = ({ stage }: { stage: Stage }) => {
  const { controllerName, isControllerDead } = useStageControllerStatus(stage);
  const stagePhase = getStagePhase(stage, isControllerDead);
  const currentPromotionPath = useCurrentPromotionRef(stage)?.path;

  if (getCurrentFreight(stage).length === 0) {
    return <>-</>;
  }

  const Comp = (
    <Flex align='center' gap={4}>
      {stagePhase}{' '}
      <StageConditionIcon
        stage={stage}
        noTooltip
        className='text-[10px]'
        isControllerDead={isControllerDead}
        controllerName={controllerName}
      />
    </Flex>
  );

  if (stagePhase === 'Promoting' && currentPromotionPath) {
    return <Link to={currentPromotionPath}>{Comp}</Link>;
  }

  return Comp;
};
