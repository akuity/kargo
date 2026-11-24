import {
  faCircle,
  faCircleNotch,
  faHeart,
  faHeartBroken,
  faInfoCircle,
  faQuestionCircle,
  IconDefinition
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';
import { CSSProperties } from 'react';

import { useToneColor } from '@ui/features/common/status-tone';
import { Health } from '@ui/gen/api/v2/models';

import { healthTone, HealthStatus, healthStatusToEnum } from './utils';

export const HealthStatusIcon = (props: {
  health?: Health;
  style?: CSSProperties;
  hideColor?: boolean;
  className?: string;
  noTooltip?: boolean;
}) => {
  const { health, hideColor } = props;
  const reason = health?.issues?.join('; ') ?? '';
  const toneColor = useToneColor(healthTone(health?.status));

  const Icon = (
    <FontAwesomeIcon
      icon={iconForHealthStatus(health)}
      spin={healthStatusToEnum(health?.status) === HealthStatus.PROGRESSING}
      className={props.className}
      style={{ color: hideColor ? undefined : toneColor, ...props.style }}
    />
  );

  if (props.noTooltip) {
    return Icon;
  }

  return (
    <Tooltip
      title={
        <>
          <div className='mb-2'>
            {health?.status && (
              <>
                <b>Health Status:</b> {health.status}
              </>
            )}
          </div>
          {reason !== '' && (
            <>
              <FontAwesomeIcon icon={faInfoCircle} className='mr-2' /> {reason}
            </>
          )}
        </>
      }
    >
      {Icon}
    </Tooltip>
  );
};

const iconForHealthStatus = (health?: Health): IconDefinition => {
  switch (healthStatusToEnum(health?.status)) {
    case HealthStatus.HEALTHY:
      return faHeart;
    case HealthStatus.UNHEALTHY:
    case HealthStatus.DEGRADED:
      return faHeartBroken;
    case HealthStatus.PROGRESSING:
      return faCircleNotch;
    case HealthStatus.UNKNOWN:
      return faQuestionCircle;
    default:
      return faCircle;
  }
};
