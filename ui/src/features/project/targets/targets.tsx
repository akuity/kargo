import { faBullseye, faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Input, Segmented, Typography } from 'antd';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { LoadingState } from '@ui/features/common/loading-state';
import { useCurrentRounds } from '@ui/features/stage/use-current-round';
import { useListStages, useListTargets } from '@ui/gen/api/v2/core/core';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { useWatchStages } from '../pipelines/use-watch-stages';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { TargetsList } from './list/targets-list';
import { Facets, labelFacets, matchesFacets, noFacets, stateFacet } from './model/target-facets';
import {
  NO_GROUPING,
  STAGE_GROUPING,
  filterGroupedRows,
  groupAxis,
  groupTargetRows,
  groupedRows,
  keptRows
} from './model/target-grouping';
import { labelKeys } from './model/target-labels';
import { behindCount, stateCounts, targetRows } from './model/target-rows';
import { matchesSearch } from './model/target-search';
import { targetAwareStages } from './model/target-stages';
import { FiltersPanel } from './shared/filters-panel';
import { GroupByFilter } from './shared/group-by-filter';
import { useFreightLabel } from './shared/use-freight-label';
import { TargetGroupCard } from './summary/group-card';
import { StatTiles } from './summary/stat-tiles';

const plural = (count: number) => (count === 1 ? '' : 's');

// The two ways to read a fleet. Cards answer what shape it is in; the list
// answers which Targets, exactly.
type View = 'cards' | 'list';
// Where a reader's choice of view and grouping is kept between visits. The
// view is one preference across every project; the grouping is per project.
const viewKey = 'targets-view';
const groupByKey = (project: string) => `targets-group-by/${project}`;

const views = [
  { value: 'cards', label: 'Cards' },
  { value: 'list', label: 'List' }
];

/**
 * Targets summarizes a project's fleet.
 *
 * The page answers two questions in the order they get asked: how is the
 * fleet as a whole, and where in the fleet is the trouble. The second is
 * answered by slicing the fleet along whichever label the reader picks --
 * region, tier, cloud, whatever the Targets happen to carry -- and drawing
 * each slice as one card. Opening a card lists its Targets.
 *
 * Governance is evaluated from each Stage's selectors, mirroring the
 * controller. What a Target runs and how healthy it is comes from its own
 * status, scoped per Stage; what is moving right now comes from each Stage's
 * current round of fan-out.
 */
export const Targets = () => {
  const { name: project = '' } = useParams();

  const stagesQuery = useListStages(project, { freightOrigins: [] });
  // Targets live in the database, which offers no watch; the list is polled.
  const targetsQuery = useListTargets(project, undefined, {
    query: { refetchInterval: TARGETS_POLL_INTERVAL_MS }
  });
  useWatchStages(project);

  const allStages = useMemo(() => stagesQuery.data?.data?.items || [], [stagesQuery.data]);
  const stages = useMemo(() => targetAwareStages(allStages), [allStages]);
  const targets = useMemo(() => targetsQuery.data?.data?.items || [], [targetsQuery.data]);
  const rounds = useCurrentRounds(project, stages);

  const freightLabel = useFreightLabel(project);

  // Which view and which grouping a reader prefers outlive the visit. The
  // search and the filters do not: a filter still in force from last week would
  // hide most of the fleet with nothing on screen to say why. Both views share
  // them, so switching view keeps the same Targets on screen.
  const [storedView, setView] = useLocalStorage<View>(viewKey, 'cards');
  // Anything may be in storage, including a view that no longer exists.
  const view: View = storedView === 'list' ? 'list' : 'cards';

  const [search, setSearch] = useState('');
  const [facets, setFacets] = useState<Facets>(noFacets());

  // Grouping is remembered per project, because label keys are a project's own:
  // the axis that slices one fleet may mean nothing in the next.
  const [storedGroupBy, setGroupBy] = useLocalStorage<string>(groupByKey(project), NO_GROUPING);
  const keys = useMemo(() => labelKeys(targets), [targets]);

  const allRows = useMemo(() => targetRows(targets, stages, rounds), [targets, stages, rounds]);

  // A remembered key that this project's Targets do not carry would group
  // everything into one unlabeled bucket, which says less than not grouping at
  // all -- so it falls back to that until the keys are loaded and do contain it.
  // Grouping by Stage goes by the same rule, and so waits for a Stage to
  // actually govern something: target-aware Stages whose selectors match
  // nothing yet would leave every Target in one "no Stage" bucket.
  const byStage = useMemo(() => allRows.some((row) => row.stages.length > 0), [allRows]);
  const groupBy =
    keys.includes(storedGroupBy) || (byStage && storedGroupBy === STAGE_GROUPING)
      ? storedGroupBy
      : NO_GROUPING;
  const groupKey = groupBy || undefined;

  const rows = useMemo(
    () => allRows.filter((row) => matchesSearch(row, search)),
    [allRows, search]
  );

  // Grouped before anything is faceted or counted, because grouped by Stage a
  // row's state is the one its own Stage sees and only grouping settles which
  // Stage that is. The facets then count, and filter, the rows the groups hold.
  const grouped = useMemo(
    () => groupTargetRows(rows, groupKey, allStages),
    [rows, groupKey, allStages]
  );
  const facetRows = useMemo(() => groupedRows(grouped), [grouped]);
  const labels = useMemo(() => labelFacets(facetRows, keys, facets), [facetRows, keys, facets]);
  const states = useMemo(() => stateFacet(facetRows, facets), [facetRows, facets]);
  // How many label values the whole fleet carries, whatever is searched or
  // ticked. The panel offers a label search only when there are enough to need
  // one, and a figure that moved with every tick would pull the search box out
  // from under the reader.
  const labelValueCount = useMemo(
    () => labelFacets(allRows, keys, noFacets()).reduce((sum, f) => sum + f.values.length, 0),
    [allRows, keys]
  );
  const groups = useMemo(
    () => filterGroupedRows(grouped, (row) => matchesFacets(row, facets)),
    [grouped, facets]
  );

  // The Targets the filters have left, one row each however the page is
  // grouped.
  const shownRows = useMemo(() => keptRows(rows, groups), [rows, groups]);
  // Fleet figures, one row per Target however the page is grouped. See
  // StatTiles for why these and the per-Stage cards differ.
  const counts = useMemo(() => stateCounts(shownRows), [shownRows]);

  const loading = stagesQuery.isLoading || targetsQuery.isLoading;

  return (
    <div className='p-6 flex flex-col gap-4 overflow-auto h-full'>
      <Flex justify='space-between' align='flex-start' gap={16} wrap>
        <Flex vertical gap={4}>
          <Typography.Title level={4} className='!mb-0'>
            <FontAwesomeIcon icon={faBullseye} className='mr-2' />
            Targets
          </Typography.Title>
          <Typography.Text type='secondary' className='text-xs'>
            {loading
              ? 'Loading the fleet'
              : `${shownRows.length} of ${targets.length} Target${plural(targets.length)}, ` +
                `governed by ${stages.length} Stage${plural(stages.length)}`}
          </Typography.Text>
        </Flex>
        <Segmented
          disabled={loading}
          value={view}
          onChange={(value) => setView(value as View)}
          options={views}
        />
      </Flex>

      <Flex gap={16} wrap align='center'>
        <Input
          allowClear
          disabled={loading}
          prefix={<FontAwesomeIcon icon={faMagnifyingGlass} className='text-gray-400' />}
          placeholder='Search name, or selector e.g. tier=primary,cloud!=azure'
          className='flex-1 min-w-[260px]'
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        {(keys.length > 0 || byStage) && (
          <GroupByFilter
            keys={keys}
            byStage={byStage}
            value={groupBy}
            onChange={setGroupBy}
            disabled={loading}
          />
        )}
      </Flex>

      {loading ? (
        <LoadingState />
      ) : (
        <Flex gap={16} align='flex-start' className='min-h-0' wrap>
          {/* With no Targets there is nothing to filter, and the panel would be
              a heading over an empty box. */}
          {targets.length > 0 && (
            <div className='w-[260px] shrink-0 max-w-full'>
              <FiltersPanel
                labels={labels}
                states={states}
                facets={facets}
                labelValueCount={labelValueCount}
                onChange={setFacets}
              />
            </div>
          )}
          <Flex vertical gap={16} className='flex-1 min-w-[320px]'>
            {view === 'list' ? (
              <TargetsList
                project={project}
                groups={groups}
                shown={shownRows.length}
                total={rows.length}
                groupKey={groupKey}
                freightLabel={freightLabel}
              />
            ) : (
              <>
                <StatTiles
                  shown={shownRows.length}
                  total={targets.length}
                  counts={counts}
                  behind={behindCount(shownRows)}
                />

                {groups.length === 0 ? (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      targets.length ? 'No Targets match' : 'This project has no Targets'
                    }
                  />
                ) : (
                  <Flex vertical gap={8}>
                    <Typography.Text strong className='text-sm'>
                      {groupKey
                        ? `${groups.length} group${plural(groups.length)} by ${groupAxis(groupKey)}`
                        : 'All Targets'}
                    </Typography.Text>
                    {/* No items-start: grid rows stretch, so cards sitting side
                        by side are the same height however much each has to
                        say. Ungrouped, the fleet is one card, and it takes the
                        full width rather than a single column's worth. */}
                    <div
                      className='grid gap-3'
                      style={{
                        gridTemplateColumns: groupKey
                          ? 'repeat(auto-fill, minmax(340px, 1fr))'
                          : '1fr'
                      }}
                    >
                      {groups.map((group) => (
                        <TargetGroupCard
                          key={group.value || 'all'}
                          project={project}
                          groupKey={groupKey}
                          group={group}
                          freightLabel={freightLabel}
                        />
                      ))}
                    </div>
                  </Flex>
                )}
              </>
            )}
          </Flex>
        </Flex>
      )}
    </div>
  );
};
