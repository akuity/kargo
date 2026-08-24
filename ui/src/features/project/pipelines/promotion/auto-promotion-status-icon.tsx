import { faBolt, faPause, faSnowflake } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tooltip } from 'antd';

import type { Stage } from '@ui/gen/api/v2/models';

import { holdStateMessage, stageHasAutoPromotionHold } from './auto-promotion';
import { isPromotionWindowClosed, promotionWindowClosedMessage } from './promotion-window';

type AutoPromotionStatusIconProps = {
  stage: Stage;
  autoPromotionEnabled: boolean;
};

export const AutoPromotionStatusIcon = ({
  stage,
  autoPromotionEnabled
}: AutoPromotionStatusIconProps) => {
  const hasHold = stageHasAutoPromotionHold(stage);

  // a closed promotion window is only worth surfacing here for Stages that would
  // otherwise promote on their own -- elsewhere the user learns of it when acting
  const windowClosed = autoPromotionEnabled && isPromotionWindowClosed(stage);

  if (!autoPromotionEnabled && !hasHold) {
    return null;
  }

  const labels: string[] = [];

  if (hasHold) {
    labels.push(holdStateMessage(stage));
  } else if (autoPromotionEnabled) {
    labels.push('Auto-promotion enabled');
  }

  if (windowClosed) {
    labels.push(promotionWindowClosedMessage(stage));
  }

  // each state gets its own line in the tooltip, but a single label for readers
  const label = labels.join(' ');
  const title = labels.map((text) => <div key={text}>{text}</div>);

  // the badge slot holds one glyph -- a hold is the more specific of the two
  const badge = hasHold ? faPause : windowClosed ? faSnowflake : undefined;

  return (
    <Tooltip title={title}>
      <span aria-label={label} className='inline-flex mr-1.5 relative'>
        <FontAwesomeIcon icon={faBolt} className='text-[10px]' />
        {badge && (
          <FontAwesomeIcon
            icon={badge}
            className='text-[7px] absolute'
            style={{ bottom: '-5px', right: '-3px' }}
          />
        )}
      </span>
    </Tooltip>
  );
};
