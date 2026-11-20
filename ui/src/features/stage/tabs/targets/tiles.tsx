import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Typography, theme } from 'antd';
import { ReactNode, useMemo } from 'react';
import { Link } from 'react-router-dom';

import { TargetRow } from '@ui/features/project/targets/model/target-rows';
import { surfaceClassName } from '@ui/features/project/targets/shared/surface';
import { Round, isRolloutRound, roundPath } from '@ui/features/stage/utils/promotion-rollout';

import { MarkBar, MarkDot } from './mark-visuals';
import { Marking, tally } from './marks';
import { healthTally, neverPromoted, notAttemptedCount, roundTally } from './summary';

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
 * what Freight is on it, how its latest round went, how healthy it is, and how
 * much of it has never been promoted to at all.
 *
 * Every figure is counted from the same rows the squares below are drawn from,
 * so the tiles and the grid can never disagree.
 */
export const TargetTiles = ({
  project,
  stageName,
  rows,
  round,
  freightMarks,
  freightLabel
}: {
  project: string;
  stageName: string;
  rows: TargetRow[];
  // The Stage's latest round, when it has had one.
  round?: Round;
  // The Freight vocabulary, shared with the grid's Freight colouring so a
  // piece of Freight is the same colour in both.
  freightMarks: Marking;
  freightLabel: (name: string) => string;
}) => {
  const { token } = theme.useToken();

  const freight = useMemo(() => tally(rows, freightMarks), [rows, freightMarks]);
  const promoted = useMemo(() => roundTally(rows), [rows]);
  const unattempted = useMemo(() => notAttemptedCount(rows), [rows]);
  const health = useMemo(() => healthTally(rows), [rows]);
  const never = useMemo(() => neverPromoted(rows), [rows]);

  const roundFreight = round?.spec?.freight;
  const roundLink = roundPath(project, stageName, round);
  // A round that never reached a Target -- blocked, or still being admitted --
  // has nothing to report per Target. The drawer explains why above the tabs.
  const reached = promoted.inRound > 0;
  // With nothing failed, the one thing worth saying of the Targets the round
  // promoted to: that some are still moving, that some were cancelled, or
  // that every Target went well -- which is not said of a round that left
  // any Target unattempted.
  const outcome =
    promoted.failed > 0
      ? undefined
      : promoted.inFlight > 0
        ? `${promoted.inFlight} in flight`
        : promoted.aborted > 0
          ? `${promoted.aborted} aborted`
          : unattempted === 0
            ? 'all succeeded'
            : undefined;

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

      <Tile heading={`Last round${roundFreight ? ` · ${freightLabel(roundFreight)}` : ''}`}>
        <Figure
          value={reached ? promoted.succeeded : '—'}
          suffix={reached ? `/ ${promoted.inRound}` : undefined}
        />
        <Typography.Text type='secondary' className='text-xs'>
          {!round ? (
            'never promoted'
          ) : !reached ? (
            'not attempted'
          ) : (
            <>
              {promoted.failed > 0 && (
                <span style={{ color: token.colorError }}>{promoted.failed} failed</span>
              )}
              {outcome}
              {/* Targets the round ended without promoting to are said apart
                  from the ones it failed on: nothing ran against them. */}
              {unattempted > 0 &&
                `${promoted.failed > 0 || outcome ? ' · ' : ''}${unattempted} not attempted`}
              {promoted.notInRound > 0 && ` · ${promoted.notInRound} joined later`}
            </>
          )}
          {/* The round has a page of its own, which opens on the Promotions
              tab with this round expanded: every child Promotion, and where
              the failures fell. */}
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
        <Figure value={never} />
        <Typography.Text type='secondary' className='text-xs'>
          {never ? 'no Freight yet' : 'all have Freight'}
        </Typography.Text>
      </Tile>
    </Flex>
  );
};
