import {
  faCircle,
  faCircleMinus,
  faCircleNotch,
  faMagnifyingGlass,
  faTimesCircle,
  faSync,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo } from 'react';

import { deadControllerReason } from '@ui/features/common/controller-status/controller-status';
import { useToneColor } from '@ui/features/common/status-tone';
import { Stage } from '@ui/gen/api/v2/models';

import styles from './styles.module.less';
import { getStageStatus, phaseTone, StagePhase } from './utils';

// The mark for each phase. The phases that describe work under way say so by
// moving.
//
// Promoting is a plain spinner rather than the animated truck it used to be.
// The truck already marks the Promote action in the node's menu, so the state
// and the action now read as two different things; and a spinner holds its
// shape at the 9-10px the pipeline sets it in, where the truck's own line
// height and drifting exhaust did not.
const phaseMark: Record<StagePhase, { icon: IconDefinition; spin?: boolean; animation?: string }> =
  {
    Promoting: { icon: faCircleNotch, spin: true },
    Verifying: { icon: faMagnifyingGlass, animation: styles.magnifyingGlass },
    Reconciling: { icon: faSync, animation: styles.rotate },
    Failed: { icon: faTimesCircle },
    Ready: { icon: faCircle },
    Unknown: { icon: faCircleMinus }
  };

// StageConditionIcon is the mark beside a Stage's phase -- or, where there is
// no room for the word, in place of it. It states the phase getStageStatus
// reports and nothing of its own.
export const StageConditionIcon = memo(
  ({
    stage,
    className,
    noTooltip,
    hideColor,
    isControllerDead,
    controllerName
  }: {
    stage: Stage;
    className?: string;
    noTooltip?: boolean;
    // Set where the mark sits beside something already stating the phase, so
    // that it takes the color of what it belongs to instead of its own.
    hideColor?: boolean;
    isControllerDead?: boolean;
    controllerName?: string;
  }) => {
    const { phase, message } = getStageStatus(stage, isControllerDead);
    const toneColor = useToneColor(phaseTone[phase]);

    // The one reason that is not the Stage's own: its conditions were written
    // by the controller that has since gone silent, which is why the phase
    // above is Failed however healthy they read.
    const reason = isControllerDead ? deadControllerReason(controllerName || '') : message;

    const mark = phaseMark[phase];

    const Icon = (
      <FontAwesomeIcon
        icon={mark.icon}
        spin={mark.spin}
        className={classNames(className, mark.animation)}
        style={{ color: hideColor ? undefined : toneColor }}
      />
    );

    if (noTooltip) {
      return Icon;
    }

    return (
      <Tooltip
        title={
          <>
            <b>Stage Status:</b> {phase}
            {reason && <div>{reason}</div>}
          </>
        }
      >
        {Icon}
      </Tooltip>
    );
  }
);
