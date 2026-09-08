import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { Link } from 'react-router-dom';

import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { Stage } from '@ui/gen/api/v2/models';

import { getCurrentPromotionRef } from './stage-meta-utils';

export const StageNodePhase = (props: { stage: Stage }) => {
  const { controllerName, isControllerDead } = useStageControllerStatus(props.stage);
  const stagePhase = getStagePhase(props.stage, isControllerDead);
  const currentPromotionPath = getCurrentPromotionRef(props.stage)?.path;

  const Phase = (
    <Flex align='center' gap={4}>
      {stagePhase}{' '}
      <StageConditionIcon
        className='text-[10px]'
        conditions={props.stage?.status?.conditions || []}
        noTooltip
        isControllerDead={isControllerDead}
        controllerName={controllerName}
      />
      {stagePhase === 'Promoting' && currentPromotionPath && (
        <FontAwesomeIcon icon={faExternalLink} className='text-[8px]' />
      )}
    </Flex>
  );

  if (stagePhase === 'Promoting' && currentPromotionPath) {
    return <Link to={currentPromotionPath}>{Phase}</Link>;
  }

  return Phase;
};
