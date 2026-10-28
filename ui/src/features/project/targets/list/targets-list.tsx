import { Button, Collapse, Empty, Flex, Tag, Typography } from 'antd';
import { useEffect, useMemo, useState } from 'react';

import { Stage } from '@ui/gen/api/v2/models';

import { TargetTable } from '../detail/target-table';
import { Facets, labelFacets, matchesFacets, noFacets, stateFacet } from '../model/target-facets';
import {
  STAGE_GROUPING,
  TargetGroup,
  filterGroupedRows,
  groupTargetRows,
  groupTitle,
  groupedRows
} from '../model/target-grouping';
import { TargetRow, byWorstState, stateCounts } from '../model/target-rows';
import { targetStatePresentations } from '../model/target-state';
import { StateBar } from '../shared/state-visuals';

import { FiltersPanel } from './filters-panel';

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
 * TargetsList is the fleet read as a list rather than as cards: a facet panel
 * that narrows it by label and by state, and the Targets themselves grouped
 * into sections, each opening on the worst of what it holds.
 *
 * It answers a different question from the cards. The cards ask what shape the
 * fleet is in; the list asks which Targets, exactly, and lets a reader cut the
 * fleet down until only those are left.
 */
export const TargetsList = ({
  project,
  rows,
  keys,
  groupKey,
  stages,
  freightLabel
}: {
  project: string;
  // The rows the search has left, before grouping or faceting.
  rows: TargetRow[];
  // Every label key the Targets carry, as facet axes.
  keys: string[];
  // The label key the list is grouped by, if any.
  groupKey?: string;
  // The Project's Stages, which order the groups when grouped by Stage.
  stages: Stage[];
  freightLabel: (name: string) => string;
}) => {
  const [facets, setFacets] = useState<Facets>(noFacets());

  // Grouped before anything is faceted or counted, because grouped by Stage a
  // row's state is the one its own Stage sees and only grouping settles which
  // Stage that is. The facets then count, and filter, the rows the groups hold.
  const grouped = useMemo(() => groupTargetRows(rows, groupKey, stages), [rows, groupKey, stages]);
  const facetRows = useMemo(() => groupedRows(grouped), [grouped]);

  const labels = useMemo(() => labelFacets(facetRows, keys, facets), [facetRows, keys, facets]);
  const states = useMemo(() => stateFacet(facetRows, facets), [facetRows, facets]);
  // Sorted after grouping, not before: grouped by Stage, a row's state is the
  // one its Stage sees, which is not the state it was sorted by beforehand.
  const groups = useMemo(
    () =>
      filterGroupedRows(grouped, (row) => matchesFacets(row, facets)).map((group) => ({
        ...group,
        rows: byWorstState(group.rows)
      })),
    [grouped, facets]
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

  // Distinct Targets, not rows: grouped by Stage a Target appears once per
  // Stage that governs it, and the caption counts Targets.
  const shown = useMemo(
    () => new Set(groupedRows(groups).map((row) => row.target.metadata?.name)).size,
    [groups]
  );

  return (
    <Flex gap={16} align='flex-start' className='min-h-0' wrap>
      <div className='w-[260px] shrink-0 max-w-full'>
        <FiltersPanel labels={labels} states={states} facets={facets} onChange={setFacets} />
      </div>
      <Flex vertical gap={8} className='flex-1 min-w-[320px]'>
        <Flex justify='space-between' align='center' gap={12} wrap>
          <Typography.Text type='secondary' className='text-xs'>
            {shown === rows.length
              ? `${rows.length} Target${rows.length === 1 ? '' : 's'}`
              : `Showing ${shown} of ${rows.length} Targets`}
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
    </Flex>
  );
};
