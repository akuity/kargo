import { Typography, theme } from 'antd';
import { useMemo } from 'react';

import { TargetRow, byWorstState } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { useStateColors } from '../shared/use-state-colors';

// A group can run to hundreds of Targets. Past this many squares the shape of
// the group is already clear and the rest only cost DOM, so the grid draws
// what it can and says how many it left out.
const maxSquares = 500;

// GridMark is how one square is painted and named: a colour, whether it is
// drawn as an outline, and the word the tooltip gives it.
export type GridMark = { label: string; color: string; hollow?: boolean };

/**
 * TargetGrid draws one small square per Target so the shape of a group is
 * legible at a glance: where the trouble is and how much of it there is.
 * Hovering one names its Target and what it is painted for.
 *
 * By default squares are coloured by state and sorted worst first, so whatever
 * needs attention is in the top-left corner. A caller colouring by something
 * else -- which Freight a Target runs, how its last promotion went -- passes
 * `mark` and sorts the rows itself, since only it knows which of its own
 * categories comes first.
 *
 * With `onSelect` a square is a button: picking one is how a reader asks about
 * the Target behind it, and the picked square carries a ring so the grid says
 * which one is being answered. Without it the squares stay inert, carrying a
 * plain title attribute -- a grid of hundreds should not mount hundreds of
 * popups to name one.
 */
export const TargetGrid = ({
  rows,
  mark,
  size = 8,
  selected,
  onSelect
}: {
  rows: TargetRow[];
  mark?: (row: TargetRow) => GridMark;
  size?: number;
  // The name of the Target whose square is picked, if any.
  selected?: string;
  onSelect?: (row: TargetRow) => void;
}) => {
  const { token } = theme.useToken();
  const colors = useStateColors();

  const sorted = useMemo(() => (mark ? rows : byWorstState(rows)), [rows, mark]);
  const shown = sorted.slice(0, maxSquares);

  const paint =
    mark ??
    ((row: TargetRow): GridMark => {
      const { label, hollow } = targetStatePresentations[row.state];
      return { label, hollow, color: colors[row.state] };
    });

  return (
    <div>
      <div className='flex flex-wrap gap-[2px]'>
        {shown.map((row) => {
          const { label, color, hollow } = paint(row);
          const name = row.target.metadata?.name || '';
          const style = {
            width: size,
            height: size,
            padding: 0,
            background: hollow ? 'transparent' : color,
            border: hollow ? `1px solid ${color}` : 'none',
            outline: name === selected ? `2px solid ${token.colorText}` : undefined,
            outlineOffset: 1
          };
          return onSelect ? (
            <button
              key={name}
              type='button'
              title={`${name} — ${label}`}
              aria-pressed={name === selected}
              onClick={() => onSelect(row)}
              className='block rounded-[2px] cursor-pointer'
              style={style}
            />
          ) : (
            <span
              key={name}
              title={`${name} — ${label}`}
              className='block rounded-[2px]'
              style={style}
            />
          );
        })}
      </div>
      {sorted.length > shown.length && (
        <Typography.Text type='secondary' className='text-xs'>
          +{sorted.length - shown.length} more not drawn
        </Typography.Text>
      )}
    </div>
  );
};
