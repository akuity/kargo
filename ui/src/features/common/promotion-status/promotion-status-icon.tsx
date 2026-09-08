import { MessageTooltip } from '@ui/features/project/pipelines/message-tooltip';
import { PromotionStatus } from '@ui/gen/api/v2/models';

import { getPromotionPhasePresentation } from './promotion-phase';

const PhaseAndMessage = ({ status, subject }: { status: PromotionStatus; subject: string }) => (
  <div>
    <div className='font-semibold'>
      {subject} {status.phase}
    </div>
    <div>{status.message}</div>
  </div>
);

export const PromotionStatusIcon = ({
  status,
  subject = 'Promotion',
  color,
  ...props
}: {
  status?: PromotionStatus;
  // subject names what the phase belongs to in the tooltip. A
  // PromotionRequest's phases share the Promotion vocabulary, so the same icon
  // presents both.
  subject?: string;
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
      message={<PhaseAndMessage status={status} subject={subject} />}
      icon={icon}
      iconColor={color ? color : iconColor}
      spin={spin}
      {...props}
    />
  );
};
