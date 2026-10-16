import { Button, Flex, Typography } from 'antd';
import { Fragment, useMemo, useState } from 'react';

import { SmallLabel } from '@ui/features/common/small-label';

import { TargetDrawer } from '../detail/target-drawer';
import { TargetGroup, groupHeading } from '../model/target-grouping';
import { freightTally, stateCounts } from '../model/target-rows';
import { FreightLink } from '../shared/freight-link';
import { StateBar, StateLegend } from '../shared/state-visuals';

import { TargetGrid } from './target-grid';

// Past a handful the footer's Freight list stops being a summary.
const maxFreightNamed = 3;

/**
 * TargetGroupCard is one slice of the fleet: its make-up as a bar, a count per
 * state, a square per Target, and the Freight it is running. Opening it lists
 * the Targets themselves in a drawer.
 */
export const TargetGroupCard = ({
  project,
  groupKey,
  group,
  freightLabel
}: {
  project: string;
  groupKey?: string;
  group: TargetGroup;
  freightLabel: (name: string) => string;
}) => {
  const [open, setOpen] = useState(false);

  const counts = useMemo(() => stateCounts(group.rows), [group.rows]);
  const freight = useMemo(() => freightTally(group.rows), [group.rows]);
  const named = freight.slice(0, maxFreightNamed);

  return (
    <div className='flex flex-col h-full rounded-md p-4 border border-solid bg-white dark:bg-neutral-800 border-gray-200 dark:border-neutral-700'>
      <Flex justify='space-between' align='flex-start' gap={12}>
        <div className='min-w-0'>
          {groupKey && <SmallLabel className='font-mono'>{groupKey}=</SmallLabel>}
          <div className='text-lg font-semibold truncate'>{groupHeading(group, groupKey)}</div>
        </div>
        <div className='text-right shrink-0'>
          <div className='text-lg font-semibold leading-tight'>{group.rows.length}</div>
          <Typography.Text type='secondary' className='text-xs'>
            targets
          </Typography.Text>
        </div>
      </Flex>

      <StateBar counts={counts} className='my-3' />
      <StateLegend counts={counts} />
      <div className='mt-3'>
        <TargetGrid rows={group.rows} />
      </div>

      <Flex justify='space-between' align='baseline' gap={12} className='mt-auto pt-3'>
        <Typography.Text type='secondary' className='text-xs flex-1 min-w-0'>
          {named.length ? 'Freight: ' : 'No Freight on these Targets yet'}
          {/* Separators sit outside the nowrap spans, so the line still has
              somewhere to break. */}
          {named.map(({ name, count }, index) => (
            <Fragment key={name}>
              {index > 0 && ' · '}
              <span className='whitespace-nowrap'>
                <FreightLink project={project} name={name} label={freightLabel(name)} /> ×{count}
              </span>
            </Fragment>
          ))}
          {freight.length > named.length && (
            <>
              {' · '}
              <span className='whitespace-nowrap'>+{freight.length - named.length} more</span>
            </>
          )}
        </Typography.Text>
        <Button
          type='link'
          size='small'
          className='px-0 h-auto shrink-0 whitespace-nowrap'
          aria-label={`Open ${group.rows.length} Target${group.rows.length === 1 ? '' : 's'}`}
          onClick={() => setOpen(true)}
        >
          Open targets
        </Button>
      </Flex>

      <TargetDrawer
        project={project}
        groupKey={groupKey}
        group={group}
        freightLabel={freightLabel}
        open={open}
        onClose={() => setOpen(false)}
      />
    </div>
  );
};
