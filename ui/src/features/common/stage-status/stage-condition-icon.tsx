import {
  faCircle,
  faCircleMinus,
  faMagnifyingGlass,
  faTimesCircle,
  faSync,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';
import { memo } from 'react';

import { useToneColor } from '@ui/features/common/status-tone';
import { Stage } from '@ui/gen/api/v2/models';

import styles from './styles.module.less';
import { TruckIcon } from './truck-icon/truck-icon';
import { getStageStatus, phaseTone, StagePhase } from './utils';

// The mark for each phase. The two that describe work under way say so by
// moving. Promoting's entry is never drawn -- it gets the truck below -- but
// the map stays total so a new phase cannot be added without a mark.
const phaseMark: Record<StagePhase, { icon: IconDefinition; animation?: string }> = {
  Promoting: { icon: faCircleMinus },
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
    const deadControllerReason = controllerName
      ? `Controller '${controllerName}' is dead or nonexistent`
      : 'The default controller is dead or nonexistent';

    const reason = isControllerDead ? deadControllerReason : message;

    const mark = phaseMark[phase];

    const Icon =
      phase === 'Promoting' ? (
        <TruckIcon className={className} />
      ) : (
        <FontAwesomeIcon
          icon={mark.icon}
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
