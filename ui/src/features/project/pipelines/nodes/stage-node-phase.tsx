import { faCircleNotch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Link } from 'react-router-dom';

import { StageConditionIcon } from '@ui/features/common/stage-status/stage-condition-icon';
import { useStageControllerStatus } from '@ui/features/common/stage-status/use-stage-controller-status';
import { getStageStatus, phaseTone } from '@ui/features/common/stage-status/utils';
import { useToneColor } from '@ui/features/common/status-tone';
import { Stage } from '@ui/gen/api/v2/models';

import './stage-node.less';

import { getCurrentPromotionRef } from './stage-meta-utils';

export const StageNodePhase = (props: { stage: Stage }) => {
  const { controllerName, isControllerDead } = useStageControllerStatus(props.stage);
  const { phase: stagePhase, message } = getStageStatus(props.stage, isControllerDead);
  const currentPromotionPath = getCurrentPromotionRef(props.stage)?.path;

  // Unknown resolves to the muted tone, which is the tone the dash below
  // wants anyway.
  const color = useToneColor(phaseTone[stagePhase]);

  // Nothing to report. A Stage with no freight has each of its conditions
  // stood down with reason NoFreight, which leaves the phase Unknown -- a word
  // and a grey icon that between them say nothing. State it as the Health
  // column beside it states an absent status, so the band reads as one thing.
  if (stagePhase === 'Unknown') {
    return (
      <span
        className='stage-node__status-value'
        style={{ color, fontWeight: 400 }}
        title='No status reported'
      >
        &mdash;
      </span>
    );
  }

  // A promotion in flight is the one phase that is still moving, so it says so
  // with a spinner. This is the only place the node reports it: the footer used
  // to say it again underneath.
  const isPromoting = stagePhase === 'Promoting';

  const Phase = (
    <span className='stage-node__status-value' style={{ color }}>
      {isPromoting ? (
        <FontAwesomeIcon icon={faCircleNotch} spin className='text-[9px]' />
      ) : (
        /* the tooltip would otherwise repeat the word next to it; it earns
           its place only when there is more to say than the word. A dead
           controller is that case without a message: getStageStatus leaves
           the reason to the icon, which names the controller itself. */
        <StageConditionIcon
          className='text-[9px]'
          stage={props.stage}
          hideColor
          noTooltip={!message && !isControllerDead}
          isControllerDead={isControllerDead}
          controllerName={controllerName}
        />
      )}
      {stagePhase}
    </span>
  );

  if (isPromoting && currentPromotionPath) {
    return <Link to={currentPromotionPath}>{Phase}</Link>;
  }

  return Phase;
};
