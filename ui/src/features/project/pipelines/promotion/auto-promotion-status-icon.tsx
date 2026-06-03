import { faBolt, faHourglassHalf, faPause } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { useMemo } from 'react';

import type { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

import {
  autoPromotionHoldStateActive,
  autoPromotionHoldStatePending,
  getAutoPromotionHoldEntries
} from './auto-promotion';

type AutoPromotionStatusIconProps = {
  stage: Stage;
  autoPromotionEnabled: boolean;
};

export const AutoPromotionStatusIcon = ({
  stage,
  autoPromotionEnabled
}: AutoPromotionStatusIconProps) => {
  const holdEntries = useMemo(() => getAutoPromotionHoldEntries(stage), [stage]);

  const hasActiveHold = holdEntries.some(
    (entry) => entry.hold.state === autoPromotionHoldStateActive
  );
  const hasPendingHold = holdEntries.some(
    (entry) => entry.hold.state === autoPromotionHoldStatePending
  );

  const icon = hasActiveHold ? faPause : hasPendingHold ? faHourglassHalf : faBolt;
  const label = hasActiveHold
    ? 'Auto-promotion paused after rollback'
    : hasPendingHold
      ? 'Rollback promotion pending. Auto-promotion will pause if it succeeds.'
      : autoPromotionEnabled
        ? 'Auto-promotion enabled'
        : 'Auto-promotion hold exists, but auto-promotion is disabled';

  return (
    <span aria-label={label} className='inline-flex mr-1'>
      <FontAwesomeIcon icon={icon} className='text-[10px]' />
    </span>
  );
};
