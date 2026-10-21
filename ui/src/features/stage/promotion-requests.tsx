import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Empty, Flex, Table, Tag, Tooltip, Typography, theme } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format, formatDistanceToNow } from 'date-fns';
import { useMemo, useState } from 'react';
import { Link, generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { PhaseFilterChips } from '@ui/features/common/promotion-status/phase-filter-chips';
import {
  PromotionPhase,
  getPromotionPhasePresentation
} from '@ui/features/common/promotion-status/promotion-phase';
import { RoundProgressBar } from '@ui/features/common/promotion-status/round-progress-bar';
import { getAlias } from '@ui/features/common/utils';
import { getPromotionActor } from '@ui/features/project/pipelines/promotion/get-promotion-actor';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import { promotionRequestCompareFn } from './utils/promotion-request';
import {
  describeTargets,
  freightVersion,
  phaseFilters,
  requestDurationMs,
  requestPhase,
  roundSummary,
  targetsTally
} from './utils/promotion-request-list';
import { formatStepDuration } from './utils/step-duration';

const requestsPagination = { hideOnSinglePage: true, pageSize: 10, showSizeChanger: false };

// TargetsCell draws how a round fared: the same bar every other surface draws a
// round with, and beneath it the figure per phase. Only what needs to stand out
// is painted -- a failure in red, movement in blue. Succeeded is the expected
// case and reads as ordinary text, and Pending and Aborted are painted with
// fills too faint to read as text at all.
const TargetsCell = ({ promotionRequest }: { promotionRequest: PromotionRequest }) => {
  const { token } = theme.useToken();
  const summary = roundSummary(promotionRequest);
  const tally = targetsTally(summary);

  const partColor = (phase: PromotionPhase) => {
    if (phase === 'Failed' || phase === 'Errored') {
      return token.colorError;
    }
    return phase === 'Running' ? token.colorInfo : token.colorTextSecondary;
  };

  if (!tally.total) {
    return (
      <Tooltip title='The Stage governed no Targets when this request was created'>
        <Typography.Text type='secondary' className='text-xs'>
          no Targets
        </Typography.Text>
      </Tooltip>
    );
  }

  return (
    <Flex vertical gap={6} className='min-w-0'>
      <RoundProgressBar summary={summary} />
      <span className='text-xs' aria-label={describeTargets(tally)}>
        {tally.parts.map(({ phase, count }, index) => (
          <span key={phase} style={{ color: partColor(phase) }}>
            {index > 0 && <span style={{ color: token.colorTextQuaternary }}> · </span>}
            {count} {phase.toLowerCase()}
          </span>
        ))}
        <Typography.Text type='secondary'> / {tally.total}</Typography.Text>
      </span>
    </Flex>
  );
};

type Props = {
  projectName: string;
  stageName: string;
};

/**
 * PromotionRequests is the Promotions tab of a target-aware Stage: the rounds
 * of promotion the Stage has run, newest first.
 *
 * Such a Stage promotes Freight to the Targets it governs by way of a
 * PromotionRequest, which fans out into one child Promotion per Target. A
 * fleet of any size therefore produces hundreds of Promotions that say nothing
 * individually, so this level shows only the requests -- how far each one got
 * and what it was promoting -- and the children are read inside a request.
 */
export const PromotionRequests = ({ projectName, stageName }: Props) => {
  const { name: routeProjectName } = useParams();
  const project = routeProjectName || projectName;
  const navigate = useNavigate();

  const pathOf = (promotionRequest: PromotionRequest) =>
    generatePath(paths.promotionRequest, {
      name: project,
      stageName,
      promotionRequestName: promotionRequest?.metadata?.name || ''
    });

  const listQuery = useListPromotionRequests(
    projectName,
    { stage: stageName },
    { query: { enabled: !!projectName && !!stageName } }
  );

  useWatchPromotionRequests(projectName, stageName, !listQuery.isLoading);

  const freightMap = useGetFreightMap(project);

  const [selectedPhase, setSelectedPhase] = useState<PromotionPhase | undefined>();

  const promotionRequests = useMemo(
    () => [...(listQuery.data?.data?.items || [])].sort(promotionRequestCompareFn),
    [listQuery.data]
  );

  const rows = useMemo(
    () =>
      selectedPhase
        ? promotionRequests.filter((request) => requestPhase(request) === selectedPhase)
        : promotionRequests,
    [promotionRequests, selectedPhase]
  );

  const columns: ColumnsType<PromotionRequest> = [
    {
      title: 'Phase',
      width: 120,
      render: (_, promotionRequest) => {
        const phase = requestPhase(promotionRequest);
        const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
        return (
          <Tag className='m-0' color={tagColor} icon={<FontAwesomeIcon icon={icon} spin={spin} />}>
            {phase}
          </Tag>
        );
      }
    },
    {
      title: 'Freight',
      render: (_, promotionRequest) => {
        const freightName = promotionRequest?.spec?.freight || '';
        const freight = freightMap[freightName];
        const version = freightVersion(freight);
        // A request that did not succeed explains itself in status.message, and
        // that explanation is the reason a reader opened the tab. One that did
        // succeed says so just as plainly, so the message is shown either way
        // and only its color changes.
        const message = promotionRequest?.status?.message;
        const failed = ['Failed', 'Errored'].includes(requestPhase(promotionRequest));
        return (
          <Flex vertical gap={2} className='min-w-0'>
            <Tooltip title={freightName}>
              <span>
                <Link className='font-semibold' to={pathOf(promotionRequest)}>
                  {getAlias(freight) || freightName.slice(0, 7)}
                </Link>
                {version && (
                  <Typography.Text type='secondary' className='ml-2 text-xs font-mono'>
                    {version}
                  </Typography.Text>
                )}
              </span>
            </Tooltip>
            <Typography.Text type='secondary' className='text-xs font-mono'>
              {promotionRequest?.metadata?.name}
            </Typography.Text>
            {message && (
              <Typography.Text type={failed ? 'danger' : 'secondary'} className='text-xs'>
                {message}
              </Typography.Text>
            )}
          </Flex>
        );
      }
    },
    {
      title: 'Targets',
      width: 230,
      render: (_, promotionRequest) => <TargetsCell promotionRequest={promotionRequest} />
    },
    {
      title: 'Started',
      width: 170,
      render: (_, promotionRequest) => {
        // A request the reconciler has not started yet has no startedAt, but it
        // does have a creation time, which is when the user asked for it.
        const started =
          parseDate(promotionRequest?.status?.startedAt) ||
          parseDate(promotionRequest?.metadata?.creationTimestamp);
        if (!started) {
          return null;
        }
        return (
          <Flex vertical gap={2}>
            <Typography.Text className='text-xs'>
              {format(started, 'MMM d, HH:mm:ss')}
            </Typography.Text>
            <Typography.Text type='secondary' className='text-xs'>
              {formatDistanceToNow(started, { addSuffix: true })}
            </Typography.Text>
          </Flex>
        );
      }
    },
    {
      title: 'Duration',
      width: 110,
      render: (_, promotionRequest) => {
        const duration = requestDurationMs(promotionRequest);
        return (
          <Typography.Text
            className='text-xs tabular-nums'
            type={duration === undefined ? 'secondary' : undefined}
          >
            {duration === undefined ? '-' : formatStepDuration(duration)}
          </Typography.Text>
        );
      }
    },
    {
      title: 'By',
      width: 110,
      render: (_, promotionRequest) => (
        <Typography.Text className='text-xs'>{getPromotionActor(promotionRequest)}</Typography.Text>
      )
    }
  ];

  return (
    <Flex vertical gap={12}>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Typography.Title level={5} className='m-0'>
          Promotion Requests
        </Typography.Title>
        {promotionRequests.length > 0 && (
          <PhaseFilterChips
            filters={phaseFilters(promotionRequests)}
            selected={selectedPhase}
            onSelect={setSelectedPhase}
          />
        )}
      </Flex>

      <Table
        dataSource={rows}
        columns={columns}
        size='small'
        pagination={requestsPagination}
        onRow={(promotionRequest) => ({
          className: 'cursor-pointer',
          onClick: (event) => {
            // The Freight cell is a real link, so that it can be opened in a
            // new tab and read before it is followed. Let it handle its own
            // click rather than navigating twice.
            if ((event.target as HTMLElement).closest('a')) {
              return;
            }
            navigate(pathOf(promotionRequest));
          }
        })}
        rowKey={(promotionRequest) => promotionRequest?.metadata?.name || ''}
        loading={listQuery.isLoading}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description='No Freight has been promoted to this Stage yet'
            />
          )
        }}
      />

      {promotionRequests.length > 0 && (
        <Typography.Text type='secondary' className='text-xs'>
          Each request fans out to one Promotion per Target. Open a request to see its Promotions.
        </Typography.Text>
      )}
    </Flex>
  );
};
