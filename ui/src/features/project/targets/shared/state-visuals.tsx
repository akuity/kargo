import { Flex, Tooltip, theme } from 'antd';
import classNames from 'classnames';

import {
  StateCounts,
  TargetState,
  describeStates,
  stateOrder,
  targetStatePresentations,
  totalStateCount
} from '../model/target-state';

import { useStateColors } from './use-state-colors';

// StateDot is one state's colour at legend size. A hollow state draws as an
// outline, so "nothing has promoted here yet" reads as an empty slot rather
// than as a status of its own.
export const StateDot = ({ state, size = 8 }: { state: TargetState; size?: number }) => {
  const colors = useStateColors();
  const { hollow } = targetStatePresentations[state];
  return (
    <span
      className='inline-block shrink-0 rounded-sm'
      style={{
        width: size,
        height: size,
        background: hollow ? 'transparent' : colors[state],
        border: hollow ? `1px solid ${colors[state]}` : undefined
      }}
    />
  );
};

// StateBar draws a tally as one bar: the share of Targets in each state, in the
// order a legend lists them. Every bar on the page -- a Stage's rollout, a
// group's make-up -- is this one component, so they all read alike.
export const StateBar = ({
  counts,
  height = 6,
  className
}: {
  counts: StateCounts;
  height?: number;
  className?: string;
}) => {
  const { token } = theme.useToken();
  const colors = useStateColors();
  const total = totalStateCount(counts);
  const description = describeStates(counts);

  return (
    <Tooltip title={description}>
      <div
        role='img'
        aria-label={description}
        className={classNames('flex overflow-hidden', className)}
        style={{ height, borderRadius: height, background: token.colorFillTertiary }}
      >
        {total > 0 &&
          stateOrder.map((state) =>
            counts[state] ? (
              <div
                key={state}
                data-state={state}
                style={{
                  width: `${(counts[state] / total) * 100}%`,
                  background: colors[state],
                  transition: 'width 0.3s ease'
                }}
              />
            ) : null
          )}
      </div>
    </Tooltip>
  );
};

// StateLegend lists what a tally holds, a dot and a count per state. States
// nothing is in are left out: a reader should not have to skip past zeroes to
// find the states that matter.
//
// With onSelect, each entry doubles as a quick filter: clicking one narrows to
// that state, clicking it again clears. Entries that are not the active one dim
// so the filter in force reads as such.
export const StateLegend = ({
  counts,
  selected,
  onSelect
}: {
  counts: StateCounts;
  selected?: TargetState;
  onSelect?: (state?: TargetState) => void;
}) => (
  <Flex gap={onSelect ? 2 : 12} wrap align='center'>
    {stateOrder.map((state) => {
      if (!counts[state]) {
        return null;
      }
      const { label, short } = targetStatePresentations[state];
      const entry = (
        <>
          <StateDot state={state} />
          <span className='text-xs whitespace-nowrap'>
            <span className='font-semibold'>{counts[state]}</span> {short}
          </span>
        </>
      );

      if (!onSelect) {
        return (
          <Flex key={state} gap={6} align='center'>
            {entry}
          </Flex>
        );
      }

      const active = selected === state;
      return (
        <Tooltip key={state} title={active ? 'Show every Target' : `Show only ${label} Targets`}>
          <button
            type='button'
            aria-pressed={active}
            onClick={() => onSelect(active ? undefined : state)}
            className={classNames(
              'flex items-center gap-1.5 rounded px-2 py-1 border-0 cursor-pointer',
              'hover:bg-gray-100 dark:hover:bg-neutral-700',
              active ? 'bg-gray-100 dark:bg-neutral-700' : 'bg-transparent',
              { 'opacity-45': selected && !active }
            )}
          >
            {entry}
          </button>
        </Tooltip>
      );
    })}
  </Flex>
);
