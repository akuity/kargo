import { Flex } from 'antd';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { getCurrentFreight } from '@ui/features/common/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

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

  if (stagePhase === 'Promoting') {
    return (
      <Link
        to={generatePath(paths.promotion, {
          name: stage?.metadata?.namespace,
          promotionId: stage?.status?.currentPromotion?.name
        })}
      >
        {Comp}
      </Link>
    );
  }

  return Comp;
};
