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
import { useListPromotionRollouts } from '@ui/gen/api/v2/core/core';
import { PromotionRollout } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { PROMOTION_ROLLOUTS_POLL_INTERVAL_MS } from './constants';
import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import {
  isPromotionRolloutPhaseTerminal,
  promotionRolloutCompareFn
} from './utils/promotion-rollout';
import {
  describeTargets,
  freightVersion,
  phaseFilters,
  rolloutDurationMs,
  rolloutPhase,
  roundSummary,
  targetsTally
} from './utils/promotion-rollout-list';
import { formatStepDuration } from './utils/step-duration';

const rolloutsPagination = { hideOnSinglePage: true, pageSize: 10, showSizeChanger: false };

// TargetsCell draws how a round fared: the same bar every other surface draws a
// round with, and beneath it the figure per phase. Only what needs to stand out
// is painted -- a failure in red, movement in blue. Succeeded is the expected
// case and reads as ordinary text, and Pending and Aborted are painted with
// fills too faint to read as text at all.
const TargetsCell = ({ promotionRollout }: { promotionRollout: PromotionRollout }) => {
  const { token } = theme.useToken();
  const summary = roundSummary(promotionRollout);
  const tally = targetsTally(summary);

  const partColor = (phase: PromotionPhase) => {
    if (phase === 'Failed' || phase === 'Errored') {
      return token.colorError;
    }
    return phase === 'Running' ? token.colorInfo : token.colorTextSecondary;
  };

  // spec.targets is absent until the reconciler has resolved the Stage's
  // selectors, and an empty list once it resolved to nothing. A rollout that
  // ended without ever resolving them is not still resolving.
  if (
    !promotionRollout?.spec?.targets &&
    !isPromotionRolloutPhaseTerminal(rolloutPhase(promotionRollout))
  ) {
    return (
      <Tooltip title='The Targets this rollout promotes to are still being resolved'>
        <Typography.Text type='secondary' className='text-xs'>
          resolving Targets
        </Typography.Text>
      </Tooltip>
    );
  }

  if (!tally.total) {
    return (
      <Tooltip title='The Stage governed no Targets when this rollout was created'>
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
 * PromotionRollouts is the Promotions tab of a target-aware Stage: the rounds
 * of promotion the Stage has run, newest first.
 *
 * Such a Stage promotes Freight to the Targets it governs by way of a
 * PromotionRollout, which fans out into one child Promotion per Target. A
 * fleet of any size therefore produces hundreds of Promotions that say nothing
 * individually, so this level shows only the rollouts -- how far each one got
 * and what it was promoting -- and the children are read inside a rollout.
 *
 * PromotionRollouts live in the database, which offers no watch, so the list
 * is polled.
 */
export const PromotionRollouts = ({ projectName, stageName }: Props) => {
  const { name: routeProjectName } = useParams();
  const project = routeProjectName || projectName;
  const navigate = useNavigate();

  const pathOf = (promotionRollout: PromotionRollout) =>
    generatePath(paths.promotionRollout, {
      name: project,
      stageName,
      promotionRolloutName: promotionRollout?.metadata?.name || ''
    });

  const listQuery = useListPromotionRollouts(
    projectName,
    { stage: stageName },
    {
      query: {
        enabled: !!projectName && !!stageName,
        refetchInterval: PROMOTION_ROLLOUTS_POLL_INTERVAL_MS
      }
    }
  );

  const freightMap = useGetFreightMap(project);

  const [selectedPhase, setSelectedPhase] = useState<PromotionPhase | undefined>();

  const promotionRollouts = useMemo(
    () => [...(listQuery.data?.data?.items || [])].sort(promotionRolloutCompareFn),
    [listQuery.data]
  );

  const rows = useMemo(
    () =>
      selectedPhase
        ? promotionRollouts.filter((rollout) => rolloutPhase(rollout) === selectedPhase)
        : promotionRollouts,
    [promotionRollouts, selectedPhase]
  );

  const columns: ColumnsType<PromotionRollout> = [
    {
      title: 'Phase',
      width: 120,
      render: (_, promotionRollout) => {
        const phase = rolloutPhase(promotionRollout);
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
      render: (_, promotionRollout) => {
        const freightName = promotionRollout?.spec?.freight || '';
        const freight = freightMap[freightName];
        const version = freightVersion(freight);
        // A rollout that did not succeed explains itself in status.message, and
        // that explanation is the reason a reader opened the tab. One that did
        // succeed says so just as plainly, so the message is shown either way
        // and only its color changes.
        const message = promotionRollout?.status?.message;
        const failed = ['Failed', 'Errored'].includes(rolloutPhase(promotionRollout));
        return (
          <Flex vertical gap={2} className='min-w-0'>
            <Tooltip title={freightName}>
              <span>
                <Link className='font-semibold' to={pathOf(promotionRollout)}>
                  {getAlias(freight) || freightName.slice(0, 7)}
                </Link>
                {version && (
                  <Typography.Text type='secondary' className='ml-2 text-xs font-mono'>
                    {version}
                  </Typography.Text>
                )}
              </span>
            </Tooltip>
            {/* A Stage numbers its rollouts like CI runs, so the number is the
                short way to refer to one. */}
            <Typography.Text type='secondary' className='text-xs font-mono'>
              {promotionRollout?.number ? `#${promotionRollout.number} · ` : ''}
              {promotionRollout?.metadata?.name}
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
      render: (_, promotionRollout) => <TargetsCell promotionRollout={promotionRollout} />
    },
    {
      title: 'Started',
      width: 170,
      render: (_, promotionRollout) => {
        // A rollout the reconciler has not started yet has no startedAt, but it
        // does have a creation time, which is when the user asked for it.
        const started =
          parseDate(promotionRollout?.status?.startedAt) ||
          parseDate(promotionRollout?.metadata?.creationTimestamp);
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
      render: (_, promotionRollout) => {
        const duration = rolloutDurationMs(promotionRollout);
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
      render: (_, promotionRollout) => (
        <Typography.Text className='text-xs'>{getPromotionActor(promotionRollout)}</Typography.Text>
      )
    }
  ];

  return (
    <Flex vertical gap={12}>
      <Flex justify='space-between' align='center' gap={16} wrap>
        <Typography.Title level={5} className='m-0'>
          Promotion Rollouts
        </Typography.Title>
        {promotionRollouts.length > 0 && (
          <PhaseFilterChips
            filters={phaseFilters(promotionRollouts)}
            selected={selectedPhase}
            onSelect={setSelectedPhase}
          />
        )}
      </Flex>

      <Table
        dataSource={rows}
        columns={columns}
        size='small'
        pagination={rolloutsPagination}
        onRow={(promotionRollout) => ({
          className: 'cursor-pointer',
          onClick: (event) => {
            // The Freight cell is a real link, so that it can be opened in a
            // new tab and read before it is followed. Let it handle its own
            // click rather than navigating twice.
            if ((event.target as HTMLElement).closest('a')) {
              return;
            }
            navigate(pathOf(promotionRollout));
          }
        })}
        rowKey={(promotionRollout) => promotionRollout?.metadata?.name || ''}
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

      {promotionRollouts.length > 0 && (
        <Typography.Text type='secondary' className='text-xs'>
          Each rollout fans out to one Promotion per Target. Open a rollout to see its Promotions.
        </Typography.Text>
      )}
    </Flex>
  );
};
