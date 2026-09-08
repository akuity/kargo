import { faBullseye, faLayerGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import {
  Collapse,
  Empty,
  Flex,
  Input,
  Select,
  Skeleton,
  Table,
  Tag,
  Tooltip,
  Typography
} from 'antd';
import { ColumnsType } from 'antd/es/table';
import { formatDistanceToNow } from 'date-fns';
import { useMemo, useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  getPromotionPhasePresentation,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getAlias } from '@ui/features/common/utils';
import { useGetFreightMap } from '@ui/features/stage/tabs/freight-history/use-get-freight-map';
import { useWatchPromotionRequests } from '@ui/features/stage/use-watch-promotion-requests';
import { blockingMessage } from '@ui/features/stage/utils/promotion-request';
import { useListPromotionRequests, useListStages, useListTargets } from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { useWatchStages } from '../pipelines/use-watch-stages';

import {
  FleetGroup,
  FleetRow,
  GROUP_BY_STAGE,
  SEVERITY_SETTLED,
  TargetStageCell,
  UNLABELED,
  fleetRows,
  groupRows,
  groupSeverity,
  labelKeys,
  requestForStage,
  rowPhaseCounts,
  targetAwareStages
} from './fleet-utils';
import { useWatchTargets } from './use-watch-targets';

import './fleet.less';

// The Select option that turns grouping off. Label keys are never empty, so
// the empty string cannot collide with a real key.
const NO_GROUPING = '';

// Fixed widths for every column but Target, which takes the remaining space.
// The group tables hide their own headers and share one header row above the
// groups, so the widths must agree between that row and the tables.
const columnWidths = { stage: 200, freight: 200, phase: 180, when: 180 };

// PhaseChips renders one chip per phase present among a set of rows. It reads
// the same per-row cells as the tables, so chips and rows can never disagree.
const PhaseChips = ({ byPhase }: { byPhase: Record<string, number> }) => (
  <Flex gap={4} wrap>
    {promotionPhases.map((phase) => {
      const count = byPhase[phase] || 0;
      if (!count) {
        return null;
      }
      const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
      const description = `${count} ${phase.toLowerCase()} Target${count === 1 ? '' : 's'}`;
      return (
        <Tooltip key={phase} title={description}>
          <Tag
            className='m-0'
            color={tagColor}
            icon={<FontAwesomeIcon icon={icon} spin={spin} />}
            aria-label={description}
          >
            {count}
          </Tag>
        </Tooltip>
      );
    })}
  </Flex>
);

const PhaseCell = ({ project, cell }: { project: string; cell: TargetStageCell }) => {
  if (!cell.governed) {
    return (
      <Tooltip title='No target-aware Stage selects this Target'>
        <Typography.Text type='secondary' className='text-xs'>
          not governed
        </Typography.Text>
      </Tooltip>
    );
  }
  if (!cell.request) {
    return (
      <Typography.Text type='secondary' className='text-xs'>
        never promoted
      </Typography.Text>
    );
  }
  if (!cell.included) {
    return (
      <Tooltip title='This Target joined the Stage after its latest round of promotion was created'>
        <Typography.Text type='secondary' className='text-xs'>
          not in latest round
        </Typography.Text>
      </Tooltip>
    );
  }
  const phase = cell.phase || 'Pending';
  const message = cell.promotion ? undefined : blockingMessage(cell.request);
  return (
    <Flex gap={8} align='center'>
      <PromotionStatusIcon status={{ phase, message }} />
      {cell.promotion ? (
        <Link
          className='text-xs'
          to={generatePath(paths.promotion, { name: project, promotionId: cell.promotion })}
        >
          {phase}
        </Link>
      ) : (
        <Tooltip title='No child Promotion was recorded for this Target in the latest round'>
          <Typography.Text className='text-xs'>{phase}</Typography.Text>
        </Tooltip>
      )}
    </Flex>
  );
};

// RoundSummary is the one-line account of a Stage's latest round: its phase,
// its Freight, and how its Targets fared.
const RoundSummary = ({
  request,
  rows,
  freightAlias
}: {
  request?: PromotionRequest;
  rows: FleetRow[];
  freightAlias: (name?: string) => string | undefined;
}) => {
  if (!request) {
    return (
      <Typography.Text type='secondary' className='text-xs'>
        never promoted
      </Typography.Text>
    );
  }
  return (
    <Flex gap={12} align='center' wrap>
      <Flex gap={6} align='center'>
        <PromotionStatusIcon
          subject='Promotion Request'
          status={{ phase: request.status?.phase, message: blockingMessage(request) }}
        />
        <Typography.Text className='text-xs'>{request.status?.phase || 'Pending'}</Typography.Text>
      </Flex>
      <Typography.Text type='secondary' className='text-xs'>
        {freightAlias(request.spec?.freight) || request.spec?.freight?.slice(0, 7)}
      </Typography.Text>
      <PhaseChips byPhase={rowPhaseCounts(rows).byPhase} />
    </Flex>
  );
};

/**
 * Fleet is the project-level view of Targets. Each row is one Target under one
 * Stage that governs it, with the Freight and Promotion phase of that Stage's
 * latest round. Rows group by Stage -- the rollout's own structure -- or by any
 * label key the Targets carry, which is the cross-Stage view: group by region
 * and eu's progress through canary and prod reads top to bottom.
 *
 * Groups with trouble sort first and open expanded; groups whose every row
 * succeeded start collapsed, so at twenty Stages the failures are what the
 * page leads with. Nothing here is authored by hand: governance comes from
 * evaluating each Stage's selectors, grouping axes are whatever labels exist,
 * and each row reads the Stage's most relevant PromotionRequest. Health and
 * verification per Target will join the row once Target status records them.
 */
export const Fleet = () => {
  const { name: project = '' } = useParams();

  const stagesQuery = useListStages(project, { freightOrigins: [] });
  const targetsQuery = useListTargets(project, undefined);
  const requestsQuery = useListPromotionRequests(project, undefined);
  useWatchStages(project);
  useWatchTargets(project, !!targetsQuery.data);
  useWatchPromotionRequests(project, undefined, !!requestsQuery.data);
  const freightMap = useGetFreightMap(project);

  const [groupBy, setGroupBy] = useState<string>();
  const [search, setSearch] = useState('');
  const [stageFilter, setStageFilter] = useState<string[]>([]);

  const stages = useMemo(
    () => targetAwareStages(stagesQuery.data?.data?.items || []),
    [stagesQuery.data]
  );
  const targets = useMemo(() => targetsQuery.data?.data?.items || [], [targetsQuery.data]);
  const requests = useMemo(() => requestsQuery.data?.data?.items || [], [requestsQuery.data]);

  const keys = useMemo(() => labelKeys(targets), [targets]);
  const groupKey = groupBy === undefined ? GROUP_BY_STAGE : groupBy || undefined;

  const allRows = useMemo(() => fleetRows(targets, stages, requests), [targets, stages, requests]);

  const rows = useMemo(() => {
    const needle = search.trim().toLowerCase();
    return allRows.filter((row) => {
      if (stageFilter.length && !stageFilter.includes(row.stage?.metadata?.name || '')) {
        return false;
      }
      if (!needle) {
        return true;
      }
      const haystack = [
        row.target.metadata?.name || '',
        row.stage?.metadata?.name || '',
        ...Object.entries(row.target.metadata?.labels || {}).map(([k, v]) => `${k}=${v}`)
      ]
        .join(' ')
        .toLowerCase();
      return haystack.includes(needle);
    });
  }, [allRows, search, stageFilter]);

  const groups = useMemo(() => groupRows(rows, groupKey), [rows, groupKey]);
  const fleetCounts = useMemo(() => rowPhaseCounts(allRows), [allRows]);

  const freightAlias = (name?: string) => (name ? getAlias(freightMap[name]) : undefined);

  if (stagesQuery.isLoading || targetsQuery.isLoading || requestsQuery.isLoading) {
    return (
      <div className='p-6'>
        <Skeleton active />
      </div>
    );
  }

  if (!stages.length) {
    return (
      <div className='p-6'>
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={
            <Typography.Text type='secondary'>
              No target-aware Stages in this project. A Stage becomes target-aware when its spec
              declares <Typography.Text code>targets.selectors</Typography.Text>.
            </Typography.Text>
          }
        />
      </div>
    );
  }

  const showStageColumn = groupKey !== GROUP_BY_STAGE;

  const columns: ColumnsType<FleetRow> = [
    {
      title: 'Target',
      key: 'target',
      render: (_, row) => (
        <Flex vertical gap={4}>
          <Typography.Text className='text-xs font-semibold'>
            <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
            {row.target.metadata?.name}
          </Typography.Text>
          <Flex gap={4} wrap>
            {Object.entries(row.target.metadata?.labels || {})
              .filter(([key]) => key !== groupKey)
              .map(([key, value]) => (
                <Tag key={key} className='m-0 text-xs'>
                  {key}={value}
                </Tag>
              ))}
          </Flex>
        </Flex>
      )
    },
    ...(showStageColumn
      ? [
          {
            title: 'Stage',
            key: 'stage',
            width: columnWidths.stage,
            render: (_: unknown, row: FleetRow) =>
              row.stage ? (
                <Link
                  className='text-xs'
                  to={generatePath(paths.stage, {
                    name: project,
                    stageName: row.stage.metadata?.name || ''
                  })}
                >
                  {row.stage.metadata?.name}
                </Link>
              ) : (
                <Typography.Text type='secondary' className='text-xs'>
                  none
                </Typography.Text>
              )
          }
        ]
      : []),
    {
      title: 'Freight',
      key: 'freight',
      width: columnWidths.freight,
      render: (_, row) => {
        const freight = row.cell.request?.spec?.freight;
        if (!freight) {
          return null;
        }
        return (
          <Tooltip title={freight}>
            <Link
              className='text-xs'
              to={generatePath(paths.freight, { name: project, freightName: freight })}
            >
              {freightAlias(freight) || freight.slice(0, 7)}
            </Link>
          </Tooltip>
        );
      }
    },
    {
      title: 'Promotion',
      key: 'phase',
      width: columnWidths.phase,
      render: (_, row) => <PhaseCell project={project} cell={row.cell} />
    },
    {
      title: 'When',
      key: 'when',
      width: columnWidths.when,
      render: (_, row) => {
        const status = row.cell.request?.status;
        const finished = parseDate(status?.finishedAt);
        const started = parseDate(status?.startedAt);
        const text = finished
          ? `finished ${formatDistanceToNow(finished, { addSuffix: true })}`
          : started
            ? `started ${formatDistanceToNow(started, { addSuffix: true })}`
            : '';
        return (
          <Typography.Text type='secondary' className='text-xs'>
            {text}
          </Typography.Text>
        );
      }
    }
  ];

  // A group's header carries what a per-Stage card used to: for a Stage, the
  // round's phase, Freight, and Target tally; for a label value, the tally.
  const groupHeader = (group: FleetGroup) => {
    const count = new Set(group.rows.map((row) => row.target.metadata?.name)).size;
    const countText = `${count} Target${count === 1 ? '' : 's'}`;
    if (groupKey === GROUP_BY_STAGE) {
      const stage = stages.find((candidate) => candidate.metadata?.name === group.value);
      if (!stage) {
        return (
          <Flex gap={12} align='center' wrap>
            <Tag className='m-0'>no governing Stage</Tag>
            <Typography.Text type='secondary' className='text-xs'>
              {countText}
            </Typography.Text>
          </Flex>
        );
      }
      return (
        <Flex gap={16} align='center' wrap>
          <Link
            to={generatePath(paths.stage, { name: project, stageName: group.value })}
            className='font-semibold'
            onClick={(e) => e.stopPropagation()}
          >
            {group.value}
          </Link>
          <Typography.Text type='secondary' className='text-xs'>
            {countText}
          </Typography.Text>
          <RoundSummary
            request={requestForStage(stage, requests)}
            rows={group.rows}
            freightAlias={freightAlias}
          />
        </Flex>
      );
    }
    const muted = group.value === UNLABELED;
    return (
      <Flex gap={12} align='center' wrap>
        <Tag color={muted ? 'default' : 'blue'} className='m-0'>
          {muted ? `no ${groupKey} label` : `${groupKey}=${group.value}`}
        </Tag>
        <Typography.Text type='secondary' className='text-xs'>
          {countText}
        </Typography.Text>
        <PhaseChips byPhase={rowPhaseCounts(group.rows).byPhase} />
      </Flex>
    );
  };

  const groupTable = (group: FleetGroup, showHeader: boolean) => (
    <Table
      dataSource={group.rows}
      columns={columns}
      size='small'
      pagination={false}
      showHeader={showHeader}
      tableLayout='fixed'
      rowKey={(row) => row.key}
    />
  );

  // Groups whose every row settled successfully start collapsed; anything with
  // trouble or still moving starts open.
  const expandedByDefault = groups
    .filter((group) => groupSeverity(group) < SEVERITY_SETTLED)
    .map((group) => group.value || 'all');

  return (
    <div className='p-6 flex flex-col gap-4 overflow-auto h-full fleet'>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Flex vertical gap={4}>
          <Typography.Title level={4} className='!mb-0'>
            <FontAwesomeIcon icon={faLayerGroup} className='mr-2' />
            Fleet
          </Typography.Title>
          <Flex gap={12} align='center' wrap>
            <Typography.Text type='secondary' className='text-xs'>
              {targets.length} Target{targets.length === 1 ? '' : 's'} across {stages.length}{' '}
              target-aware Stage{stages.length === 1 ? '' : 's'}
            </Typography.Text>
            <PhaseChips byPhase={fleetCounts.byPhase} />
          </Flex>
        </Flex>
        <Flex gap={8} wrap className='fleetToolbar'>
          <Input.Search
            allowClear
            placeholder='Filter Targets by name, Stage, or label'
            className='w-72'
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <Select
            className='w-48'
            value={groupKey ?? NO_GROUPING}
            onChange={(value) => setGroupBy(value)}
            options={[
              { value: GROUP_BY_STAGE, label: 'Group by Stage' },
              ...keys.map((key) => ({ value: key, label: `Group by ${key}` })),
              { value: NO_GROUPING, label: 'No grouping' }
            ]}
          />
          <Select
            mode='multiple'
            allowClear
            className='min-w-48'
            placeholder='All Stages'
            value={stageFilter}
            onChange={setStageFilter}
            options={stages.map((stage) => ({
              value: stage.metadata?.name || '',
              label: stage.metadata?.name
            }))}
          />
        </Flex>
      </Flex>

      {groups.length === 0 && (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='No Targets match' />
      )}

      {groups.length > 0 && !groupKey && groupTable(groups[0], true)}

      {groups.length > 0 && groupKey && (
        <>
          {/* One header row for every group's table, aligned by the shared
              fixed column widths. */}
          <div className='fleetColumnHeader'>
            <span className='flex-1'>Target</span>
            {showStageColumn && <span style={{ width: columnWidths.stage }}>Stage</span>}
            <span style={{ width: columnWidths.freight }}>Freight</span>
            <span style={{ width: columnWidths.phase }}>Promotion</span>
            <span style={{ width: columnWidths.when }}>When</span>
          </div>
          <Collapse
            key={groupKey}
            defaultActiveKey={expandedByDefault}
            size='small'
            className='fleetGroups'
            items={groups.map((group) => ({
              key: group.value || 'all',
              label: groupHeader(group),
              children: groupTable(group, false)
            }))}
          />
        </>
      )}
    </div>
  );
};
