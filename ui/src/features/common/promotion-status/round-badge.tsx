import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Tag, Tooltip } from 'antd';
import { useMemo } from 'react';

import { PromotionRequestSummary } from '@ui/gen/api/v2/models';

import { getPromotionPhasePresentation } from './promotion-phase';
import { describeRound, roundBadgePhase, roundBadgeText, roundSegments } from './round-progress';

// RoundBadge is the compact sibling of RoundProgressBar: the same round, said
// as a figure instead of drawn. It shows how many Targets have succeeded over
// how many there are, colored by the worst phase present, so "37/40" in red
// tells a reader at a glance that something did not succeed. It belongs where
// a bar has no room to draw -- a table row, a graph node, a project card --
// and shares the bar's tooltip and phase vocabulary so the two never disagree.
export const RoundBadge = ({
  summary,
  className
}: {
  summary?: PromotionRequestSummary;
  className?: string;
}) => {
  const progress = useMemo(() => roundSegments(summary), [summary]);
  const phase = roundBadgePhase(progress);
  if (!phase) {
    return null;
  }
  const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
  const description = describeRound(progress);
  return (
    <Tooltip title={description}>
      <Tag
        className={className ?? 'm-0'}
        color={tagColor}
        icon={<FontAwesomeIcon icon={icon} spin={spin} />}
        aria-label={description}
      >
        {roundBadgeText(progress)}
      </Tag>
    </Tooltip>
  );
};
