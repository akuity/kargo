import { faBullseye, faCircleMinus, faX } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Empty, Flex, Input, Skeleton, Space, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { formatDistanceToNow } from 'date-fns';
import { useMemo, useState } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getAlias } from '@ui/features/common/utils';
import { useListTargets } from '@ui/gen/api/v2/core/core';
import { PromotionRequest, Stage } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { blockingMessage, roundBlock } from '../../utils/promotion-request';
import { useGetFreightMap } from '../freight-history/use-get-freight-map';

import {
  FleetRow,
  fleetRows,
  freightNames,
  matchesTarget,
  rowPhase,
  rowsSummary
} from './fleet-utils';
import { RoundCard } from './round-card';
import { useWatchTargets } from './use-watch-targets';

type Props = {
  projectName: string;
  // The Stage whose Targets are listed. Its name scopes the Target list and
  // keys each Target's per-Stage status; its current Freight collection is
  // what a Target is compared against to be called behind.
  stage: Stage;
  // The Stage's latest round, resolved once by the drawer, which also uses it
  // to explain a blocked round above the tabs. Passing it in keeps one
  // PromotionRequest fetch and watch per drawer rather than one per tab.
  round?: PromotionRequest;
};

const PromotionCell = ({
  project,
  row,
  blocked
}: {
  project: string;
  row: FleetRow;
  blocked: boolean;
}) => {
  // A blocked round never reached any Target. The drawer explains why above
  // the tabs, so the row says only that nothing was attempted, in grey rather
  // than the red the request's own phase would paint.
  if (blocked) {
    return (
      <Flex gap={8} align='center'>
        <FontAwesomeIcon icon={faCircleMinus} className='text-gray-300' />
        <Typography.Text type='secondary' className='text-xs'>
          Not attempted
        </Typography.Text>
      </Flex>
    );
  }
  if (!row.request) {
    return (
      <Typography.Text type='secondary' className='text-xs'>
        never promoted
      </Typography.Text>
    );
  }
  if (!row.included) {
    return (
      <Tooltip title='This Target joined the Stage after its latest round of promotion was created'>
        <Typography.Text type='secondary' className='text-xs'>
          not in latest round
        </Typography.Text>
      </Tooltip>
    );
  }
  const phase = row.phase || 'Pending';
  return (
    <Flex gap={8} align='center'>
      <PromotionStatusIcon
        status={{ phase, message: row.promotion ? undefined : blockingMessage(row.request) }}
      />
      {row.promotion ? (
        <Link
          className='text-xs'
          to={generatePath(paths.promotion, { name: project, promotionId: row.promotion })}
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

/**
 * Fleet lists the Targets a target-aware Stage governs: what each is running
 * for this Stage, whether it is healthy, and how the Stage's latest round of
 * promotion went for it. The bar at the top summarizes that round.
 *
 * Governance comes from the API's `stage` filter, which evaluates the Stage's
 * selectors exactly as the controller does. Freight and health per Target come
 * from the Target's own status for this Stage; the round's outcome comes from
 * the Stage's current PromotionRequest, which the drawer resolves once and
 * passes in.
 */
export const Fleet = ({ projectName, stage, round }: Props) => {
  const stageName = stage.metadata?.name || '';

  const targetsQuery = useListTargets(projectName, { stage: stageName });
  useWatchTargets(projectName, stageName, !!targetsQuery.data);
  const freightMap = useGetFreightMap(projectName);

  const rows = useMemo(
    () => fleetRows(stage, targetsQuery.data?.data?.items || [], round),
    [stage, targetsQuery.data, round]
  );

  const freightLabel = (name: string) => getAlias(freightMap[name]) || name.slice(0, 7);

  // The Promotion column's filter is controlled so the round card's chips can
  // drive it: a chip selects exactly one phase, while the header menu may
  // select several. Both write here, and the header reflects either.
  const [phaseFilter, setPhaseFilter] = useState<string[] | null>(null);
  const selectedPhase = phaseFilter?.length === 1 ? phaseFilter[0] : undefined;

  // While the round is blocked the drawer explains why above the tabs. The tab
  // then shows which Targets the Stage governs and nothing that would restate
  // the failure per Target: no round card, no phase chips, grey rows.
  const blocked = !!roundBlock(round);

  if (targetsQuery.isLoading) {
    return <Skeleton active />;
  }

  // Filter options are the phases and health states actually present, with
  // counts, so a reader never picks an option that matches nothing.
  const countBy = (key: (row: FleetRow) => string) =>
    rows.reduce<Record<string, number>>((acc, row) => {
      const k = key(row);
      acc[k] = (acc[k] || 0) + 1;
      return acc;
    }, {});
  const phaseFilters = Object.entries(countBy(rowPhase)).map(([phase, count]) => ({
    text: `${phase} (${count})`,
    value: phase
  }));
  const healthFilters = Object.entries(countBy((row) => row.health?.status || '')).map(
    ([status, count]) => ({ text: `${status || 'Not assessed'} (${count})`, value: status })
  );

  const columns: ColumnsType<FleetRow> = [
    {
      title: 'Target',
      key: 'target',
      // A free-text filter on the Target's name and labels, in the column
      // header as the other filterable tables do it.
      filterDropdown: ({ setSelectedKeys, selectedKeys, confirm, clearFilters }) => (
        <Space style={{ padding: 8 }}>
          <Space.Compact>
            <Input
              placeholder='Name or label'
              value={selectedKeys[0] as string}
              onChange={(e) => setSelectedKeys(e.target.value ? [e.target.value] : [])}
              onPressEnter={() => confirm()}
              style={{ display: 'block' }}
            />
            <Button type='primary' onClick={() => confirm()}>
              Filter
            </Button>
          </Space.Compact>
          <Button
            type='text'
            icon={<FontAwesomeIcon size='xs' icon={faX} />}
            size='small'
            onClick={() => clearFilters?.({ closeDropdown: true, confirm: true })}
          />
        </Space>
      ),
      onFilter: (value, row) => matchesTarget(row, String(value)),
      render: (_, row) => (
        <Flex vertical gap={4}>
          <Typography.Text className='text-xs font-semibold'>
            <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
            {row.target.metadata?.name}
          </Typography.Text>
          <Flex gap={4} wrap>
            {Object.entries(row.target.metadata?.labels || {}).map(([key, value]) => (
              <Tag key={key} className='m-0 text-xs'>
                {key}={value}
              </Tag>
            ))}
          </Flex>
        </Flex>
      )
    },
    {
      title: 'Freight',
      key: 'freight',
      width: 220,
      render: (_, row) => {
        // What the Target is running, from its own status; until a Promotion
        // has succeeded, the Freight the round is promoting.
        const names = row.currentFreight
          ? freightNames(row.currentFreight)
          : [row.request?.spec?.freight || ''].filter(Boolean);
        if (!names.length) {
          return (
            <Typography.Text type='secondary' className='text-xs'>
              none
            </Typography.Text>
          );
        }
        return (
          <Flex gap={6} align='center' wrap>
            {names.map((name) => (
              <Tooltip key={name} title={name}>
                <Link
                  className='text-xs'
                  to={generatePath(paths.freight, { name: projectName, freightName: name })}
                >
                  {freightLabel(name)}
                </Link>
              </Tooltip>
            ))}
            {row.upToDate === false && (
              <Tooltip title="Behind the Stage's current Freight">
                <Tag className='m-0 text-xs'>behind</Tag>
              </Tooltip>
            )}
          </Flex>
        );
      }
    },
    ...(blocked
      ? []
      : [
          {
            title: 'Health',
            key: 'health',
            width: 140,
            filters: healthFilters,
            onFilter: (value: boolean | React.Key, row: FleetRow) =>
              (row.health?.status || '') === value,
            render: (_: unknown, row: FleetRow) =>
              row.health?.status ? (
                <Flex gap={6} align='center'>
                  <HealthStatusIcon health={row.health} className='text-xs' />
                  <Typography.Text className='text-xs'>{row.health.status}</Typography.Text>
                </Flex>
              ) : (
                <Typography.Text type='secondary' className='text-xs'>
                  -
                </Typography.Text>
              )
          } satisfies ColumnsType<FleetRow>[number]
        ]),
    {
      title: 'Promotion',
      key: 'promotion',
      width: 170,
      filters: blocked ? undefined : phaseFilters,
      filteredValue: phaseFilter,
      onFilter: (value, row) => rowPhase(row) === value,
      render: (_, row) => <PromotionCell project={projectName} row={row} blocked={blocked} />
    },
    ...(blocked
      ? []
      : [
          {
            title: 'When',
            key: 'when',
            width: 170,
            render: (_: unknown, row: FleetRow) => {
              const status = row.request?.status;
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
          } satisfies ColumnsType<FleetRow>[number]
        ])
  ];

  const summary = rowsSummary(rows);

  return (
    <Flex vertical gap={16}>
      {round && !blocked ? (
        <RoundCard
          projectName={projectName}
          round={round}
          // Drawn from the rows' own phases rather than the request's
          // summary, so the card and the table below it always agree.
          summary={summary}
          freightLabel={freightLabel}
          selectedPhase={selectedPhase}
          onSelectPhase={(phase) => setPhaseFilter(phase ? [phase] : null)}
        />
      ) : (
        <Typography.Text type='secondary' className='text-xs'>
          {rows.length} Target{rows.length === 1 ? '' : 's'} governed by this Stage
          {round ? '' : ', never promoted'}
        </Typography.Text>
      )}
      {rows.length === 0 ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="No Targets match this Stage's selectors"
        />
      ) : (
        <Table
          dataSource={rows}
          columns={columns}
          size='small'
          pagination={false}
          rowKey={(row) => row.target.metadata?.name || ''}
          onChange={(_, filters) => {
            const phases = filters.promotion as string[] | null | undefined;
            setPhaseFilter(phases?.length ? phases : null);
          }}
        />
      )}
    </Flex>
  );
};
