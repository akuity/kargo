import {
  faCancel,
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { theme } from 'antd';

import { MessageTooltip } from '@ui/features/project/pipelines/message-tooltip';
import { PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

const PhaseAndMessage = ({ status }: { status: PromotionStatus }) => (
  <div>
    <div className='font-semibold'>Promotion {status.phase}</div>
    <div>{status.message}</div>
  </div>
);

export const PromotionStatusIcon = ({
  status,
  color,
  ...props
}: {
  status?: PromotionStatus;
  placement?: 'right' | 'top';
  color?: string;
  size?: 'lg' | '1x';
}) => {
  if (!status) {
    return null;
  }
  const message = <PhaseAndMessage status={status} />;
  let icon = faHourglassStart;
  let defaultColor = 'aaa';
  let spin = false;
  switch (status?.phase) {
    case 'Succeeded':
      icon = faCircleCheck;
      defaultColor = theme.defaultSeed.colorSuccess;
      break;
    case 'Failed':
    case 'Errored':
      icon = faCircleExclamation;
      defaultColor = theme.defaultSeed.colorError;
      break;
    case 'Running':
      icon = faCircleNotch;
      spin = true;
      break;
    case 'Aborted':
      icon = faCancel;
      break;
    case 'Pending':
    default:
      break;
  }

  return (
    <MessageTooltip
      message={message}
      icon={icon}
      iconColor={color ? color : defaultColor}
      spin={spin}
      {...props}
    />
  );
};
