import { MessageTooltip } from '@ui/features/project/pipelines/message-tooltip';
import { PromotionStatus } from '@ui/gen/api/v2/models';

import { getPromotionPhasePresentation } from './promotion-phase';

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
  const { icon, iconColor, spin } = getPromotionPhasePresentation(status.phase);

  return (
    <MessageTooltip
      message={<PhaseAndMessage status={status} />}
      icon={icon}
      iconColor={color ? color : iconColor}
      spin={spin}
      {...props}
    />
  );
};
