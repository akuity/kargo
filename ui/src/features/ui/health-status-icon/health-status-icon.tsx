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

import type { Health } from '@ui/gen/v1alpha1/generated_pb';

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

const iconForHealthStatus = (status?: string): IconDefinition => {
  switch (status) {
    case 'Healthy':
      return faHeart;
    case 'Unhealthy':
      return faHeartBroken;
    case 'Unknown':
      return faQuestionCircle;
    default:
      return faCircle;
  }
};

const colorForHealthStatus = (status?: string): string => {
  switch (status) {
    case 'Healthy':
      return '#52c41a';
    case 'Unhealthy':
      return '#f5222d';
    case 'Unknown':
      return '#faad14';
    default:
      return '#ccc';
  }
};
