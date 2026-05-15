import { faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStagePhase } from '@ui/features/common/stage-status/utils';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

export const StageNodePhase = (props: { stage: Stage }) => {
  const projectName = props.stage?.metadata?.namespace || '';
  const { controllerName, isControllerDead } = useStageControllerStatus(props.stage);
  const stagePhase = getStagePhase(props.stage, isControllerDead);

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
      {stagePhase === 'Promoting' && (
        <FontAwesomeIcon icon={faExternalLink} className='text-[8px]' />
      )}
    </Flex>
  );

  if (stagePhase === 'Promoting') {
    return (
      <Link
        to={generatePath(paths.promotion, {
          name: projectName,
          promotionId: props.stage?.status?.currentPromotion?.name || ''
        })}
      >
        {Phase}
      </Link>
    );
  }

  return Phase;
};
