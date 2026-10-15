import { faBullseye, faMagnifyingGlass } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Input, Segmented, Typography } from 'antd';
import { useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';

import { LoadingState } from '@ui/features/common/loading-state';
import { useCurrentRounds } from '@ui/features/stage/use-current-round';
import { getColors } from '@ui/features/stage/utils';
import { useListStages, useListTargets } from '@ui/gen/api/v2/core/core';
import { useLocalStorage } from '@ui/utils/use-local-storage';

import { useWatchStages } from '../pipelines/use-watch-stages';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { TargetsList } from './list/targets-list';
import { NO_GROUPING, groupTargetRows } from './model/target-grouping';
import { labelKeys } from './model/target-labels';
import { behindCount, stageRollouts, stateCounts, targetRows } from './model/target-rows';
import { matchesSearch } from './model/target-search';
import { targetAwareStages } from './model/target-stages';
import { GroupByFilter } from './shared/group-by-filter';
import { useFreightLabel } from './shared/use-freight-label';
import { TargetGroupCard } from './summary/group-card';
import { StageRollouts } from './summary/stage-rollout';
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
 * The page answers three questions in the order they get asked: how is the
 * fleet as a whole, how far has each Stage got in reaching the Targets it
 * governs, and where in the fleet is the trouble. The last is answered by
 * slicing the fleet along whichever label the reader picks -- region, tier,
 * cloud, whatever the Targets happen to carry -- and drawing each slice as one
 * card. Opening a card lists its Targets.
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

  const stageColorMap = useMemo(() => getColors(project, allStages), [project, allStages]);
  const freightLabel = useFreightLabel(project);

  // Which view and which grouping a reader prefers outlive the visit. The
  // search does not: a filter still in force from last week would hide most of
  // the fleet with nothing on screen to say why.
  const [storedView, setView] = useLocalStorage<View>(viewKey, 'cards');
  // Anything may be in storage, including a view that no longer exists.
  const view: View = storedView === 'list' ? 'list' : 'cards';

  const [search, setSearch] = useState('');

  // Grouping is remembered per project, because label keys are a project's own:
  // the axis that slices one fleet may mean nothing in the next.
  const [storedGroupBy, setGroupBy] = useLocalStorage<string>(groupByKey(project), NO_GROUPING);
  const keys = useMemo(() => labelKeys(targets), [targets]);
  // A remembered key that this project's Targets do not carry would group
  // everything into one unlabeled bucket, which says less than not grouping at
  // all -- so it falls back to that until the keys are loaded and do contain it.
  const groupBy = keys.includes(storedGroupBy) ? storedGroupBy : NO_GROUPING;
  const groupKey = groupBy || undefined;

  const allRows = useMemo(() => targetRows(targets, stages, rounds), [targets, stages, rounds]);
  const rows = useMemo(
    () => allRows.filter((row) => matchesSearch(row, search)),
    [allRows, search]
  );
  const groups = useMemo(() => groupTargetRows(rows, groupKey), [rows, groupKey]);
  const counts = useMemo(() => stateCounts(rows), [rows]);
  const rollouts = useMemo(() => stageRollouts(rows, allStages), [rows, allStages]);

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
              : `${rows.length} of ${targets.length} Target${plural(targets.length)}, ` +
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
        {keys.length > 0 && (
          <GroupByFilter keys={keys} value={groupBy} onChange={setGroupBy} disabled={loading} />
        )}
      </Flex>

      {loading ? (
        <LoadingState />
      ) : view === 'list' ? (
        <TargetsList
          project={project}
          rows={rows}
          keys={keys}
          groupKey={groupKey}
          freightLabel={freightLabel}
        />
      ) : (
        <>
          <StatTiles
            shown={rows.length}
            total={targets.length}
            counts={counts}
            behind={behindCount(rows)}
          />

          {rollouts.length > 0 && (
            <Flex vertical gap={8}>
              <Typography.Text strong className='text-sm'>
                Rollout by Stage
              </Typography.Text>
              <StageRollouts
                project={project}
                rollouts={rollouts}
                stageColorMap={stageColorMap}
                freightLabel={freightLabel}
              />
            </Flex>
          )}

          {groups.length === 0 ? (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={targets.length ? 'No Targets match' : 'This project has no Targets'}
            />
          ) : (
            <Flex vertical gap={8}>
              <Typography.Text strong className='text-sm'>
                {groupKey
                  ? `${groups.length} group${plural(groups.length)} by ${groupKey}`
                  : 'All Targets'}
              </Typography.Text>
              {/* No items-start: grid rows stretch, so cards sitting side by
                  side are the same height however much each has to say. */}
              <div
                className='grid gap-3'
                style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))' }}
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
    </div>
  );
};
