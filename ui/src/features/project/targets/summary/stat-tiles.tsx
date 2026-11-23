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
 * Up to date, In flight, Failed, Not attempted and Never promoted, together
 * with the Targets whose last Promotion succeeded on Freight the Stage has
 * since moved on from, partition the fleet and so sum to its size. Behind
 * Stage cuts across them: a Target whose last Promotion failed, or that a
 * rollout never attempted, is usually on older Freight than its Stage too, and
 * both readings matter -- which is why it is counted again rather than taken
 * out of another figure.
 *
 * None of these is health. They say how promotion to each Target went, which
 * is all that is recorded; a Target whose last Promotion succeeded is up to
 * date, not known to be healthy.
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
  const inFlight = counts.Promoting + counts.Pending;
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
        heading={targetStatePresentations.UpToDate.label}
        dot='UpToDate'
        value={counts.UpToDate}
        caption={share(counts.UpToDate, shown)}
      />
      <Tile
        heading='In flight'
        dot='Promoting'
        value={inFlight}
        caption={
          inFlight
            ? `${counts.Promoting} promoting · ${counts.Pending} pending`
            : 'no promotion in flight'
        }
      />
      <Tile
        heading={targetStatePresentations.Failed.label}
        dot='Failed'
        value={counts.Failed}
        caption='last promotion failed'
      />
      <Tile
        heading={targetStatePresentations.NotAttempted.label}
        dot='NotAttempted'
        value={counts.NotAttempted}
        caption='left out by a rollout'
      />
      <Tile
        heading={targetStatePresentations.NeverPromoted.label}
        dot='NeverPromoted'
        value={counts.NeverPromoted}
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
