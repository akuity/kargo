import { faBullseye } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Flex, Typography } from 'antd';
import { useState } from 'react';

import { TargetGroup, groupTitle } from '../model/target-grouping';
import { stateCounts } from '../model/target-rows';
import { TargetState } from '../model/target-state';
import { StateBar, StateLegend } from '../shared/state-visuals';

import { TargetTable } from './target-table';

const pageSize = 10;

/**
 * TargetDrawer lists the Targets of one group: what each is, which Stages
 * govern it, what it is running, and where it stands. The cards show the shape
 * of a group; this is where the Targets themselves can be read and followed
 * through to their Stages and Freight.
 */
export const TargetDrawer = ({
  project,
  groupKey,
  group,
  freightLabel,
  open,
  onClose
}: {
  project: string;
  // The label key the page is grouped by, if any.
  groupKey?: string;
  group: TargetGroup;
  freightLabel: (name: string) => string;
  open: boolean;
  onClose: () => void;
}) => {
  const counts = stateCounts(group.rows);

  // The quick filters above the table are the only way to narrow it, so the
  // state they hold is the filter itself rather than a mirror of a column's.
  const [selected, setSelected] = useState<TargetState>();
  const rows = selected ? group.rows.filter((row) => row.state === selected) : group.rows;

  return (
    <Drawer
      open={open}
      onClose={onClose}
      // Reopening a group should show all of it, so the filter does not outlive
      // the drawer that set it.
      afterOpenChange={(opened) => !opened && setSelected(undefined)}
      size='80%'
      title={
        <Flex vertical gap={4} className='font-normal'>
          <Typography.Title level={4} style={{ margin: 0 }}>
            <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
            {groupTitle(group, groupKey)}
          </Typography.Title>
          <Typography.Text type='secondary' className='text-xs'>
            {group.rows.length} Target{group.rows.length === 1 ? '' : 's'}
          </Typography.Text>
        </Flex>
      }
    >
      <Flex vertical gap={12}>
        <StateBar counts={counts} />
        <Flex justify='space-between' align='center' gap={12} wrap>
          <Typography.Text type='secondary' className='text-xs'>
            {rows.length === group.rows.length
              ? `${group.rows.length} Target${group.rows.length === 1 ? '' : 's'}`
              : `Showing ${rows.length} of ${group.rows.length} Targets`}
          </Typography.Text>
          <StateLegend counts={counts} selected={selected} onSelect={setSelected} />
        </Flex>
        <TargetTable
          // Remount on each opening, so a group is never met half-expanded
          // from the last time it was looked at.
          key={String(open)}
          project={project}
          rows={rows}
          groupKey={groupKey}
          freightLabel={freightLabel}
          pagination={{
            pageSize,
            hideOnSinglePage: true,
            showSizeChanger: false,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} Targets`
          }}
        />
      </Flex>
    </Drawer>
  );
};
