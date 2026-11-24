import { Button, Collapse, Empty, Flex, Tag, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import { TargetTable } from '../detail/target-table';
import { STAGE_GROUPING, TargetGroup, groupTitle } from '../model/target-grouping';
import { byWorstState, stateCounts } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { StateBar } from '../shared/state-visuals';

// A group opens on a preview of its worst Targets. Reading past that is a
// deliberate act, because a group may hold hundreds. Ungrouped, the one group
// is the whole list, and a preview of it would only hide the rest.
const previewSize = 6;
const fullPageSize = 20;

// GroupHeader is what a collapsed group says for itself: how big it is, what
// it is made of, and whether any of it is in trouble.
const GroupHeader = ({ group, groupKey }: { group: TargetGroup; groupKey?: string }) => {
  const counts = stateCounts(group.rows);
  return (
    <Flex align='center' gap={16} className='min-w-0'>
      {/* Monospace reads a label group for what it is, a key=value token. Only
          a label axis produces one: the Stage axis names a Stage, and no axis
          at all heads the group "All Targets". Both are prose. */}
      <span
        className={`text-sm font-semibold shrink-0 ${
          groupKey && groupKey !== STAGE_GROUPING ? 'font-mono' : ''
        }`}
      >
        {groupTitle(group, groupKey)}
      </span>
      <Typography.Text type='secondary' className='text-xs shrink-0'>
        {group.rows.length} target{group.rows.length === 1 ? '' : 's'}
      </Typography.Text>
      {/* A fixed width, not a stretch: the bar is a shape to compare group
          against group, and bars of different lengths cannot be compared. It
          still shrinks when the row runs out of room. */}
      <StateBar counts={counts} className='w-[220px] min-w-[80px]' />
      {counts.Unhealthy > 0 && (
        <Tag color='error' className='m-0 text-xs shrink-0'>
          {counts.Unhealthy} {targetStatePresentations.Unhealthy.short}
        </Tag>
      )}
    </Flex>
  );
};

const GroupBody = ({
  project,
  group,
  groupKey,
  freightLabel
}: {
  project: string;
  group: TargetGroup;
  groupKey?: string;
  freightLabel: (name: string) => string;
}) => {
  const [expanded, setExpanded] = useState(false);
  const all = expanded || !groupKey;
  const shown = all ? group.rows : group.rows.slice(0, previewSize);

  return (
    <Flex vertical gap={8}>
      <TargetTable
        project={project}
        rows={shown}
        groupKey={groupKey}
        freightLabel={freightLabel}
        pagination={
          all && group.rows.length > fullPageSize
            ? {
                pageSize: fullPageSize,
                showSizeChanger: false,
                // The pager says where in the group the page is; the caption
                // below would otherwise claim every row is on screen.
                showTotal: (total, range) => `${range[0]}-${range[1]} of ${total}`
              }
            : false
        }
      />
      {groupKey && group.rows.length > previewSize && (
        <Flex justify='space-between' align='center' gap={12} wrap>
          <Typography.Text type='secondary' className='text-xs'>
            {all ? '' : `Showing ${shown.length} of ${group.rows.length}`}
          </Typography.Text>
          <Button type='link' size='small' className='px-0' onClick={() => setExpanded(!expanded)}>
            {all
              ? `Show the first ${previewSize}`
              : `Show all ${group.rows.length} in ${groupTitle(group, groupKey)}`}
          </Button>
        </Flex>
      )}
    </Flex>
  );
};

/**
 * TargetsList is the fleet read as a list rather than as cards: the Targets
 * grouped into sections, each opening on the worst of what it holds.
 *
 * It answers a different question from the cards. The cards ask what shape the
 * fleet is in; the list asks which Targets, exactly, and the filters beside it
 * cut the fleet down until only those are left.
 */
export const TargetsList = ({
  project,
  groups: filtered,
  shown,
  total,
  groupKey,
  freightLabel
}: {
  project: string;
  // The groups the search and the filters have left.
  groups: TargetGroup[];
  // Targets the filters have left, and how many the search left before them.
  shown: number;
  total: number;
  // The label key the list is grouped by, if any.
  groupKey?: string;
  freightLabel: (name: string) => string;
}) => {
  // Sorted after grouping, not before: grouped by Stage, a row's state is the
  // one its Stage sees, which is not the state it was sorted by beforehand.
  const groups = useMemo(
    () => filtered.map((group) => ({ ...group, rows: byWorstState(group.rows) })),
    [filtered]
  );

  // Which groups are open, once the reader has said. Until then it is
  // undefined rather than empty, because those mean different things: nothing
  // chosen yet falls back to opening the first group, so that a list does not
  // start entirely shut, while an empty choice is a reader who closed them all
  // and means it.
  const [open, setOpen] = useState<string[]>();
  const openKeys = open ?? (groups.length ? [groups[0].value || 'all'] : []);

  // Changing the axis builds a different set of groups, so a choice made about
  // the old ones no longer refers to anything. Default the new ones open again
  // rather than leaving the reader a wall of shut rows.
  useEffect(() => setOpen(undefined), [groupKey]);

  return (
    <Flex vertical gap={8}>
      <Flex justify='space-between' align='center' gap={12} wrap>
        <Typography.Text type='secondary' className='text-xs'>
          {shown === total
            ? `${total} Target${total === 1 ? '' : 's'}`
            : `Showing ${shown} of ${total} Targets`}
        </Typography.Text>
        <Typography.Text type='secondary' className='text-xs'>
          Sorted worst health first
        </Typography.Text>
      </Flex>
      {groups.length === 0 ? (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No Targets match' />
      ) : (
        <Collapse
          activeKey={openKeys}
          onChange={(keys) => setOpen(Array.isArray(keys) ? keys.map(String) : [String(keys)])}
          items={groups.map((group) => ({
            key: group.value || 'all',
            label: <GroupHeader group={group} groupKey={groupKey} />,
            children: (
              <GroupBody
                project={project}
                group={group}
                groupKey={groupKey}
                freightLabel={freightLabel}
              />
            )
          }))}
        />
      )}
    </Flex>
  );
};
