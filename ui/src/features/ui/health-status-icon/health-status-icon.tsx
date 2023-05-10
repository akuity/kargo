import {
  faHeart,
  faHeartBroken,
  faQuestionCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { CSSProperties } from 'react';

export const HealthStatusIcon = (props: { health: HealthStatus; style?: CSSProperties }) => {
  const { health } = props;
  return (
    <Tooltip title={health?.statusReason}>
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

interface HealthStatus {
  status: string;
  statusReason: string;
}

const iconForHealthStatus = (status: string): IconDefinition => {
  switch (status) {
    case 'Healthy':
      return faHeart;
    case 'Unhealthy':
      return faHeartBroken;
    case 'Unknown':
      return faQuestionCircle;
    default:
      return faQuestionCircle;
  }
};

const colorForHealthStatus = (status: string): string => {
  switch (status) {
    case 'Healthy':
      return '#52c41a';
    case 'Unhealthy':
      return '#f5222d';
    case 'Unknown':
      return '#faad14';
    default:
      return '#faad14';
  }
};
