import {
  faCircle,
  faCircleMinus,
  faMagnifyingGlass,
  faTimesCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import classNames from 'classnames';

import styles from './styles.module.less';
import { TruckIcon } from './truck-icon/truck-icon';
import { StagePhase } from './utils';

export const StagePhaseIcon = (props: { phase: StagePhase; className?: string }) => {
  let icon: IconDefinition = faCircleMinus;

  switch (props.phase) {
    case StagePhase.Steady:
      icon = faCircle;
      break;

    case StagePhase.Verifying:
      icon = faMagnifyingGlass;
      break;

    case StagePhase.Failed:
      icon = faTimesCircle;
      break;
  }

  return (
    <Tooltip
      title={
        <>
          <b>Stage phase:</b> {props.phase}
        </>
      }
    >
      {props.phase === StagePhase.Promoting ? (
        <>
          <TruckIcon />
        </>
      ) : (
        <FontAwesomeIcon
          icon={icon}
          className={classNames(props.className, {
            'text-gray-400': props.phase === StagePhase.NotApplicable,
            'text-red-400': props.phase === StagePhase.Failed,
            'text-green-400': props.phase === StagePhase.Steady,
            'text-blue-500': props.phase === StagePhase.Verifying,
            [styles.magnifyingGlass]: props.phase === StagePhase.Verifying
          })}
        />
      )}
    </Tooltip>
  );
};
