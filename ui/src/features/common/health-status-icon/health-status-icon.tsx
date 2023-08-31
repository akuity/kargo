import {
  faCircle,
  faHeart,
  faHeartBroken,
  faQuestionCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { CSSProperties } from 'react';

import { Health, HealthState } from '@ui/gen/v1alpha1/types_pb';

export const HealthStatusIcon = (props: { health?: Health; style?: CSSProperties }) => {
  const { health } = props;

  return (
    <Tooltip title={health?.status}>
      <FontAwesomeIcon
        icon={iconForHealthStatus(health?.status)}
        style={{
          color: colorForHealthStatus(health?.status),
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
    case HealthState.UNKNOWN:
      return '#faad14';
    default:
      return '#ccc';
  }
};
