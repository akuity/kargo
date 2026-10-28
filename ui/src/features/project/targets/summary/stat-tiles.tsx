import { Flex, Typography } from 'antd';
import { ReactNode } from 'react';

import { StateCounts, TargetState, targetStatePresentations } from '../model/target-state';
import { StateDot } from '../shared/state-visuals';
import { surfaceClassName } from '../shared/surface';

const tileClassName = `flex-1 min-w-[132px] px-4 py-3 ${surfaceClassName}`;

const Tile = ({
  heading,
  dot,
  value,
  caption
}: {
  heading: string;
  dot?: TargetState;
  value: number;
  caption: string;
}) => (
  <div className={tileClassName}>
    <Flex gap={6} align='center'>
      {dot && <StateDot state={dot} />}
      <span className='text-xs text-gray-500 dark:text-neutral-400'>{heading}</span>
    </Flex>
    <div className='text-2xl font-semibold leading-tight mt-1'>{value}</div>
    <Typography.Text type='secondary' className='text-xs'>
      {caption}
    </Typography.Text>
  </div>
);

// share reads a count back as a percentage of the fleet, to one decimal place,
// as the figure is only meaningful next to the hundreds it is drawn from.
const share = (count: number, total: number): string =>
  total ? `${((count / total) * 100).toFixed(1)}%` : '—';

/**
 * StatTiles summarizes the Targets the page is showing.
 *
 * Healthy, Progressing, Unhealthy, Unknown and Not promoted partition the
 * fleet and so sum to its size. Behind Stage cuts across them: a Target left on
 * older Freight than its Stage may be perfectly healthy on it, and both
 * readings matter -- which is why it is counted again rather than taken out of
 * the Healthy figure.
 *
 * Targets are not health-checked yet, so for most fleets these figures come
 * from how each Stage's latest round of promotion went rather than from
 * recorded health. See governingStageState.
 *
 * These stay fleet figures however the page is grouped: a Target governed by
 * several Stages counts once, under its worst state, while the cards below
 * count each Stage's own reading. The two answer different questions.
 */
export const StatTiles = ({
  shown,
  total,
  counts,
  behind
}: {
  // Targets the search has left on the page, and how many the project holds.
  shown: number;
  total: number;
  counts: StateCounts;
  behind: number;
}): ReactNode => {
  const healthy = counts.Healthy + counts.Behind;
  return (
    <Flex gap={12} wrap>
      <div className={tileClassName}>
        <span className='text-xs text-gray-500 dark:text-neutral-400'>Targets</span>
        <div className='text-2xl font-semibold leading-tight mt-1'>{shown}</div>
        <Typography.Text type='secondary' className='text-xs'>
          of {total} in project
        </Typography.Text>
      </div>
      <Tile
        heading={targetStatePresentations.Healthy.label}
        dot='Healthy'
        value={healthy}
        caption={share(healthy, shown)}
      />
      <Tile
        heading={targetStatePresentations.Progressing.label}
        dot='Progressing'
        value={counts.Progressing}
        caption='promotion in flight'
      />
      <Tile
        heading={targetStatePresentations.Unhealthy.label}
        dot='Unhealthy'
        value={counts.Unhealthy}
        caption={share(counts.Unhealthy, shown)}
      />
      <Tile
        heading={targetStatePresentations.Unknown.label}
        dot='Unknown'
        value={counts.Unknown}
        caption='health undetermined'
      />
      <Tile
        heading={targetStatePresentations.NotPromoted.label}
        dot='NotPromoted'
        value={counts.NotPromoted}
        caption='no Freight yet'
      />
      <Tile
        heading={targetStatePresentations.Behind.label}
        dot='Behind'
        value={behind}
        caption='older Freight than Stage'
      />
    </Flex>
  );
};
