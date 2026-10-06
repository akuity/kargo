import { Tooltip, theme } from 'antd';
import classNames from 'classnames';
import { useMemo } from 'react';

import { PromotionRequestSummary } from '@ui/gen/api/v2/models';

import { getPromotionPhasePresentation } from './promotion-phase';
import { describeRound, roundSegments } from './round-progress';

type Size = 'compact' | 'full';

// RoundProgressBar draws a round of fan-out as one bar: how much of the round
// succeeded, how much failed, and how much is still moving. Every surface that
// summarizes a round -- a Stage's list row, its drawer, its pipeline node, a
// project card -- uses this one component so they all read alike.
//
// The compact size fits in a table cell or a card footer. The full size is for
// headers and adds a "succeeded / total" figure beside the bar. Hovering either
// shows the count per phase.
export const RoundProgressBar = ({
  summary,
  size = 'compact',
  className
}: {
  summary?: PromotionRequestSummary;
  size?: Size;
  className?: string;
}) => {
  const { token } = theme.useToken();
  const progress = useMemo(() => roundSegments(summary), [summary]);

  // Each phase's color is the theme token its presentation names, so the bar
  // and the Promotion status icons and chips share one vocabulary.
  const colorFor = (phase: string) => token[getPromotionPhasePresentation(phase).colorToken];

  const height = size === 'full' ? 8 : 4;
  const succeeded = progress.segments.find((s) => s.phase === 'Succeeded')?.count ?? 0;
  const description = describeRound(progress);

  return (
    <Tooltip title={description}>
      <div
        className={classNames('flex items-center gap-2 min-w-0', className)}
        role='img'
        aria-label={description}
      >
        <div
          className='flex flex-1 overflow-hidden'
          style={{ height, borderRadius: height, background: token.colorFillTertiary }}
        >
          {progress.segments.map((segment) => (
            <div
              key={segment.phase}
              data-phase={segment.phase}
              style={{
                width: `${segment.percent}%`,
                background: colorFor(segment.phase),
                transition: 'width 0.3s ease'
              }}
            />
          ))}
        </div>
        {size === 'full' && progress.total > 0 && (
          <span className='text-xs whitespace-nowrap' style={{ color: token.colorTextSecondary }}>
            {succeeded} / {progress.total}
          </span>
        )}
      </div>
    </Tooltip>
  );
};
