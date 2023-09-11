import {
  faCircle,
  faCircleNotch,
  faHeart,
  faHeartBroken,
  faQuestionCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { CSSProperties } from 'react';

import { Health, HealthState } from '@ui/gen/v1alpha1/types_pb';

import { healthStateToString } from './utils';

export const HealthStatusIcon = (props: {
  health?: Health;
  style?: CSSProperties;
  hideColor?: boolean;
}) => {
  const { health, hideColor } = props;
  const reason = health?.issues?.join('; ') ?? '';

  return (
    <Tooltip title={healthStateToString(health?.status) + (reason !== '' ? `: ${reason}` : '')}>
      <FontAwesomeIcon
        icon={iconForHealthStatus(health?.status)}
        spin={health?.status === HealthState.PROGRESSING}
        style={{
          color: !hideColor ? colorForHealthStatus(health?.status) : undefined,
          fontSize: '18px',
          ...props.style
        }}
      />
    </Tooltip>
  );
};

const iconForHealthStatus = (status?: HealthState): IconDefinition => {
  switch (status) {
    case HealthState.HEALTHY:
      return faHeart;
    case HealthState.UNHEALTHY:
      return faHeartBroken;
    case HealthState.PROGRESSING:
      return faCircleNotch;
    case HealthState.UNKNOWN:
      return faQuestionCircle;
    default:
      return faCircle;
  }
};

const colorForHealthStatus = (status?: HealthState): string => {
  switch (status) {
    case HealthState.HEALTHY:
      return '#52c41a';
    case HealthState.UNHEALTHY:
      return '#f5222d';
    case HealthState.PROGRESSING:
      return '#0dabea';
    case HealthState.UNKNOWN:
      return '#faad14';
    default:
      return '#ccc';
  }
};
