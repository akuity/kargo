import { faBolt, faPause } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';

import type { Stage } from '@ui/gen/api/v2/models';

import { holdStateMessage, stageHasAutoPromotionHold } from './auto-promotion';

type AutoPromotionStatusIconProps = {
  stage: Stage;
  autoPromotionEnabled: boolean;
};

export const AutoPromotionStatusIcon = ({
  stage,
  autoPromotionEnabled
}: AutoPromotionStatusIconProps) => {
  const hasHold = stageHasAutoPromotionHold(stage);

  if (!autoPromotionEnabled && !hasHold) {
    return null;
  }

  const label = hasHold ? holdStateMessage(stage) : 'Auto-promotion enabled';

  return (
    <span title={label} aria-label={label} className='inline-flex mr-1.5 relative'>
      <FontAwesomeIcon icon={faBolt} className='text-[10px]' />
      {hasHold && (
        <FontAwesomeIcon
          icon={faPause}
          className='text-[7px] absolute'
          style={{ bottom: '-5px', right: '-3px' }}
        />
      )}
    </span>
  );
};
