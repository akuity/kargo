import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Typography, theme } from 'antd';
import { Fragment, ReactNode, useMemo } from 'react';
import { Link } from 'react-router-dom';

import { TargetRow, stateCounts } from '@ui/features/project/targets/model/target-rows';
import { totalStateCount } from '@ui/features/project/targets/model/target-state';
import { Figures } from '@ui/features/project/targets/model/target-status';
import { surfaceClassName } from '@ui/features/project/targets/shared/surface';
import { Round, isRolloutRound, roundPath } from '@ui/features/stage/utils/promotion-rollout';

import { MarkBar, MarkDot } from './mark-visuals';
import { Marking, tally } from './marks';
import { healthTally, neverPromotedNote } from './summary';

const tileClassName = `flex-1 min-w-[180px] px-4 py-3 ${surfaceClassName}`;

const Tile = ({ heading, children }: { heading: ReactNode; children: ReactNode }) => (
  <div className={tileClassName}>
    <div className='text-xs text-gray-500 dark:text-neutral-400'>{heading}</div>
    {children}
  </div>
);

// Figure is a tile's headline number, with whatever qualifies it kept small
// beside it: "130 / 136", "0 assessed".
const Figure = ({ value, suffix }: { value: ReactNode; suffix?: ReactNode }) => (
  <div className='text-2xl font-semibold leading-tight mt-1'>
    {value}
    {suffix && (
      <Typography.Text type='secondary' className='text-sm font-normal ml-1'>
        {suffix}
      </Typography.Text>
    )}
  </div>
);

/**
 * TargetTiles answers, in one row, the four questions a Stage's fleet raises:
 * what Freight is on it, how much of it is up to date and what is keeping the
 * rest, how healthy it is when anything has said, and how much of it has never
 * been promoted to at all.
 *
 * The counts are the server's for this Stage, read by the same rules as the
 * squares below, so the two agree.
 */
export const TargetTiles = ({
  project,
  stageName,
  rows,
  figures,
  round,
  stageFreight,
  freightMarks,
  freightLabel
}: {
  project: string;
  stageName: string;
  rows: TargetRow[];
  // The Stage's counts by state, from the server.
  figures: Figures;
  // The Stage's latest round, when it has had one, which the tile links to.
  round?: Round;
  // The Freight the Stage itself is on: what up to date means.
  stageFreight: string[];
  // The Freight vocabulary, shared with the grid's Freight colouring so a
  // piece of Freight is the same colour in both.
  freightMarks: Marking;
  freightLabel: (name: string) => string;
}) => {
  const { token } = theme.useToken();

  const freight = useMemo(() => tally(rows, freightMarks), [rows, freightMarks]);
  const health = useMemo(() => healthTally(rows), [rows]);
  // A Stage the summary has not caught up with yet is counted from the rows,
  // which are read by the same rules.
  const counts = useMemo(
    () => (figures.total ? figures.counts : stateCounts(rows)),
    [figures, rows]
  );
  const total = totalStateCount(counts);
  const inFlight = counts.Promoting + counts.Pending;

  const roundLink = roundPath(project, stageName, round);
  // What keeps the rest from being up to date, worst first, each only when
  // there is any of it.
  const reasons = [
    ...(counts.Failed ? [{ text: `${counts.Failed} failed`, color: token.colorError }] : []),
    ...(inFlight ? [{ text: `${inFlight} in flight` }] : []),
    ...(counts.NotAttempted ? [{ text: `${counts.NotAttempted} not attempted` }] : []),
    ...(counts.Behind ? [{ text: `${counts.Behind} behind` }] : [])
  ];

  return (
    <Flex gap={12} wrap>
      <Tile heading='Freight on Targets'>
        <MarkBar tallies={freight} className='mt-3 mb-2' />
        <Flex gap={10} wrap align='center'>
          {freight.map(({ mark, count }) => (
            <Flex key={mark.key} gap={5} align='center'>
              <MarkDot mark={mark} />
              <span className='text-xs whitespace-nowrap'>
                <span className='font-semibold'>{count}</span> {mark.label}
              </span>
            </Flex>
          ))}
        </Flex>
      </Tile>

      <Tile
        heading={`Up to date${stageFreight.length ? ` · ${stageFreight.map(freightLabel).join(' + ')}` : ''}`}
      >
        <Figure value={counts.UpToDate} suffix={`/ ${total}`} />
        <Typography.Text type='secondary' className='text-xs'>
          {reasons.length
            ? reasons.map(({ text, color }, index) => (
                <Fragment key={text}>
                  {index > 0 && ' · '}
                  <span style={color ? { color } : undefined}>{text}</span>
                </Fragment>
              ))
            : counts.NeverPromoted
              ? 'the rest never promoted'
              : 'all up to date'}
          {/* The latest round has a page of its own: every child Promotion,
              and where the failures fell. */}
          {roundLink && (
            <>
              {' · '}
              <Link to={roundLink}>
                view {isRolloutRound(round) ? 'rollout' : 'request'}
                <FontAwesomeIcon icon={faArrowRight} className='ml-1' />
              </Link>
            </>
          )}
        </Typography.Text>
      </Tile>

      {/* Nothing assesses Target health unless the Stage's promotion steps
          produce health checks, and a tile reading "0 assessed" every time is
          noise rather than news. */}
      {health.assessed > 0 && (
        <Tile heading='Health'>
          <Figure value={health.assessed} suffix='assessed' />
          <Typography.Text type='secondary' className='text-xs'>
            {[
              ...health.byStatus.map(({ status, count }) => `${count} ${status}`),
              ...(health.unassessed ? [`${health.unassessed} not assessed`] : [])
            ].join(' · ')}
          </Typography.Text>
        </Tile>
      )}

      <Tile heading='Never promoted'>
        <Figure value={counts.NeverPromoted} />
        <Typography.Text type='secondary' className='text-xs'>
          {neverPromotedNote(counts.NeverPromoted, rows)}
        </Typography.Text>
      </Tile>
    </Flex>
  );
};
