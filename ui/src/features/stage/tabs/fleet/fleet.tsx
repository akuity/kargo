import { faBullseye } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Skeleton, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { formatDistanceToNow } from 'date-fns';
import { useMemo } from 'react';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { HealthStatusIcon } from '@ui/features/common/health-status/health-status-icon';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { RoundProgressBar } from '@ui/features/common/promotion-status/round-progress-bar';
import { getAlias } from '@ui/features/common/utils';
import { useListTargets } from '@ui/gen/api/v2/core/core';
import { Stage } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { useCurrentRound } from '../../use-current-round';
import { blockingMessage } from '../../utils/promotion-request';
import { useGetFreightMap } from '../freight-history/use-get-freight-map';

import { FleetRow, fleetRows, freightNames, rowsSummary } from './fleet-utils';
import { useWatchTargets } from './use-watch-targets';

type Props = {
  projectName: string;
  stage: Stage;
};

const PromotionCell = ({ project, row }: { project: string; row: FleetRow }) => {
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
 * the Stage's current PromotionRequest, shared with the Promotions tab.
 */
export const Fleet = ({ projectName, stage }: Props) => {
  const stageName = stage.metadata?.name || '';

  const targetsQuery = useListTargets(projectName, { stage: stageName });
  useWatchTargets(projectName, stageName, !!targetsQuery.data);
  const round = useCurrentRound(projectName, stage);
  const freightMap = useGetFreightMap(projectName);

  const rows = useMemo(
    () => fleetRows(stage, targetsQuery.data?.data?.items || [], round),
    [stage, targetsQuery.data, round]
  );

  const freightLabel = (name: string) => getAlias(freightMap[name]) || name.slice(0, 7);

  if (targetsQuery.isLoading) {
    return <Skeleton active />;
  }

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
    {
      title: 'Health',
      key: 'health',
      width: 140,
      render: (_, row) =>
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
    },
    {
      title: 'Promotion',
      key: 'promotion',
      width: 170,
      render: (_, row) => <PromotionCell project={projectName} row={row} />
    },
    {
      title: 'When',
      key: 'when',
      width: 170,
      render: (_, row) => {
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
    }
  ];

  return (
    <Flex vertical gap={16}>
      {round ? (
        <Flex gap={16} align='center' wrap>
          <RoundProgressBar
            // Drawn from the rows' own phases rather than the request's
            // summary, so the bar and the table below it always agree.
            summary={rowsSummary(rows)}
            size='full'
            className='w-96'
          />
          <Typography.Text type='secondary' className='text-xs'>
            {rows.length} Target{rows.length === 1 ? '' : 's'}
          </Typography.Text>
        </Flex>
      ) : (
        <Typography.Text type='secondary' className='text-xs'>
          {rows.length} Target{rows.length === 1 ? '' : 's'}, never promoted
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
        />
      )}
    </Flex>
  );
};
