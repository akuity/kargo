import { Typography } from 'antd';
import { useMemo } from 'react';

import { TargetRow, byWorstState } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { useStateColors } from '../shared/use-state-colors';

// A group can run to hundreds of Targets. Past this many squares the shape of
// the group is already clear and the rest only cost DOM, so the grid says how
// many it left out and the table below lists them all.
const maxSquares = 500;

/**
 * TargetGrid draws one small square per Target, coloured by its state, so the
 * shape of a group is legible at a glance: where the trouble is and how much of
 * it there is. Squares sort worst first, so whatever needs attention is always
 * in the top-left corner. Hovering one names its Target and its state.
 */
export const TargetGrid = ({ rows }: { rows: TargetRow[] }) => {
  const colors = useStateColors();

  const sorted = useMemo(() => byWorstState(rows), [rows]);
  const shown = sorted.slice(0, maxSquares);

  return (
    <div>
      <div className='flex flex-wrap gap-[2px]'>
        {shown.map((row) => {
          const { label, hollow } = targetStatePresentations[row.state];
          const name = row.target.metadata?.name || '';
          return (
            <span
              key={name}
              title={`${name} — ${label}`}
              className='block rounded-[2px] w-2 h-2'
              style={{
                background: hollow ? 'transparent' : colors[row.state],
                border: hollow ? `1px solid ${colors[row.state]}` : undefined
              }}
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
