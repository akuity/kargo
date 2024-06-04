import {
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip, theme } from 'antd';

import { PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

const PhaseAndMessage = ({ status }: { status: PromotionStatus }) => (
  <div>
    <div className='font-semibold'>Promotion {status.phase}</div>
    <div>{status.message}</div>
  </div>
);

export const PromotionStatusIcon = ({
  status,
  placement = 'right',
  color,
  size = 'lg'
}: {
  status?: PromotionStatus;
  placement?: 'right' | 'top';
  color?: string;
  size?: 'lg' | '1x';
}) => {
  switch (status?.phase) {
    case 'Succeeded':
      return (
        <Tooltip title={<PhaseAndMessage status={status} />} placement={placement}>
          <FontAwesomeIcon
            color={color ? color : theme.defaultSeed.colorSuccess}
            icon={faCircleCheck}
            size={size}
          />
        </Tooltip>
      );
    case 'Failed':
    case 'Errored':
      return (
        <Tooltip title={<PhaseAndMessage status={status} />} placement={placement}>
          <FontAwesomeIcon
            color={color ? color : theme.defaultSeed.colorError}
            icon={faCircleExclamation}
            size={size}
          />
        </Tooltip>
      );
    case 'Running':
      return (
        <Tooltip title='Promotion Running' placement={placement}>
          <FontAwesomeIcon icon={faCircleNotch} spin size={size} />
        </Tooltip>
      );
    case 'Pending':
    default:
      return (
        <Tooltip title='Promotion Pending' placement={placement}>
          <FontAwesomeIcon color={color ? color : 'aaa'} icon={faHourglassStart} size={size} />
        </Tooltip>
      );
  }
};
