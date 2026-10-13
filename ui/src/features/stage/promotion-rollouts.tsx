import { faBullseye } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';
import { useMemo, useState } from 'react';
import { Link, generatePath, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import {
  getPromotionPhasePresentation,
  promotionPhases
} from '@ui/features/common/promotion-status/promotion-phase';
import { PromotionStatusIcon } from '@ui/features/common/promotion-status/promotion-status-icon';
import { getAlias, getShortFreightLabel } from '@ui/features/common/utils';
import { useListPromotionRollouts } from '@ui/gen/api/v2/core/core';
import { PromotionRollout, PromotionRolloutSummary } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { PROMOTION_ROLLOUTS_POLL_INTERVAL_MS } from './constants';
import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import {
  PromotionRolloutTargetRow,
  isPromotionRolloutPhaseTerminal,
  promotionRolloutCompareFn,
  targetRows
} from './utils/promotion-rollout';

// summaryKey maps a phase to its counter in status.summary, whose fields are
// the lower-cased phase names.
const summaryKey = (phase: string) => phase.toLowerCase() as keyof PromotionRolloutSummary;

// TargetSummary compresses status.summary into one chip per non-zero phase. A
// rollout the controller has not yet acted on has no summary, in which case
// every Target it names is, in effect, pending.
const TargetSummary = ({ promotionRollout }: { promotionRollout: PromotionRollout }) => {
  // spec.targets is absent until the reconciler has resolved the Stage's
  // selectors, and an empty list once it resolved to nothing.
  if (!promotionRollout?.spec?.targets) {
    return (
      <Tooltip title='The Targets this rollout promotes to are still being resolved'>
        <Typography.Text type='secondary' className='text-xs'>
          resolving Targets
        </Typography.Text>
      </Tooltip>
    );
  }
  const targetCount = promotionRollout.spec.targets.length;
  if (!targetCount) {
    return (
      <Tooltip title='The Stage governed no Targets when this rollout was created'>
        <Typography.Text type='secondary' className='text-xs'>
          no Targets
        </Typography.Text>
      </Tooltip>
    );
  }

  const summary = promotionRollout?.status?.summary;

  return (
    <Flex gap={4} wrap>
      {promotionPhases.map((phase) => {
        const { icon, tagColor, spin } = getPromotionPhasePresentation(phase);
        const count =
          summary?.[summaryKey(phase)] ?? (!summary && phase === 'Pending' ? targetCount : 0);
        if (!count) {
          return null;
        }
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
};

const TargetsTable = ({
  promotionRollout,
  onSelectPromotion
}: {
  promotionRollout: PromotionRollout;
  onSelectPromotion: (name: string) => void;
}) => {
  const columns: ColumnsType<PromotionRolloutTargetRow> = [
    {
      title: 'Target',
      dataIndex: 'name',
      render: (name: string) => (
        <Typography.Text className='text-xs'>
          <FontAwesomeIcon icon={faBullseye} className='mr-2 text-gray-400' />
          {name}
        </Typography.Text>
      )
    },
    {
      title: 'Promotion',
      render: (_, target) =>
        target.promotion ? (
          <a className='text-xs font-mono' onClick={() => onSelectPromotion(target.promotion!)}>
            {target.promotion}
          </a>
        ) : (
          <Typography.Text type='secondary' className='text-xs'>
            not created yet
          </Typography.Text>
        )
    },
    {
      title: 'Phase',
      width: 140,
      render: (_, target) => (
        <Flex gap={8} align='center'>
          <PromotionStatusIcon status={{ phase: target.phase || 'Pending' }} />
          <Typography.Text className='text-xs'>{target.phase || 'Pending'}</Typography.Text>
        </Flex>
      )
    }
  ];

  return (
    <Table
      dataSource={targetRows(promotionRollout)}
      columns={columns}
      size='small'
      pagination={false}
      rowKey={(target) => target.name}
    />
  );
};

type Props = {
  projectName: string;
  stageName: string;
};

/**
 * PromotionRollouts lists the PromotionRollouts belonging to a target-aware
 * Stage, newest first, each expandable to the Targets it names and the child
 * Promotion promoting Freight to each.
 *
 * A target-aware Stage promotes Freight to the Targets it governs by way of a
 * PromotionRollout rather than a Promotion, so for such a Stage the Promotions
 * table alone would show nothing at all -- and a user who just dragged Freight
 * onto the Stage would be left with no indication of what became of it. This
 * surfaces the rollout and, when it is not going to progress, the reason.
 */
export const PromotionRollouts = ({ projectName, stageName }: Props) => {
  const { name: routeProjectName } = useParams();

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

  const freightMap = useGetFreightMap(routeProjectName || projectName);

  const [selectedPromotion, setSelectedPromotion] = useState<string | undefined>();

  const promotionRollouts = useMemo(
    () => [...(listQuery.data?.data?.items || [])].sort(promotionRolloutCompareFn),
    [listQuery.data]
  );

  if (!promotionRollouts.length) {
    return null;
  }

  const columns: ColumnsType<PromotionRollout> = [
    {
      title: 'Phase',
      width: 140,
      render: (_, promotionRollout) => {
        const phase = promotionRollout?.status?.phase;
        return (
          <Flex gap={8} align='center'>
            <PromotionStatusIcon
              subject='Rollout'
              status={{ phase: phase || 'Pending', message: promotionRollout?.status?.message }}
            />
            <Tag className='m-0' color={getPromotionPhasePresentation(phase).tagColor}>
              {phase || 'Pending'}
            </Tag>
          </Flex>
        );
      }
    },
    {
      title: 'Name',
      render: (_, promotionRollout) => (
        <Flex gap={8} align='center'>
          {!!promotionRollout?.number && (
            <Typography.Text type='secondary' className='text-xs'>
              #{promotionRollout.number}
            </Typography.Text>
          )}
          <Typography.Text copyable className='text-xs font-mono'>
            {promotionRollout?.metadata?.name}
          </Typography.Text>
        </Flex>
      )
    },
    {
      title: 'Freight',
      width: 160,
      render: (_, promotionRollout) => {
        const freightName = promotionRollout?.spec?.freight || '';
        return (
          <Link
            to={generatePath(paths.freight, {
              name: routeProjectName || projectName,
              freightName
            })}
          >
            {getShortFreightLabel(freightName, getAlias(freightMap[freightName]))}
          </Link>
        );
      }
    },
    {
      title: 'Targets',
      width: 220,
      render: (_, promotionRollout) => <TargetSummary promotionRollout={promotionRollout} />
    },
    {
      title: 'Created',
      width: 180,
      render: (_, promotionRollout) => {
        const date = parseDate(promotionRollout?.metadata?.creationTimestamp);
        return (
          <Typography.Text className='text-xs'>
            {date ? format(date, 'MMM do yyyy HH:mm:ss') : ''}
          </Typography.Text>
        );
      }
    }
  ];

  return (
    <div className='mb-6'>
      <Typography.Title level={5} className='mb-2'>
        Rollouts
      </Typography.Title>
      <Table
        dataSource={promotionRollouts}
        columns={columns}
        size='small'
        pagination={{ hideOnSinglePage: true, pageSize: 5 }}
        rowKey={(promotionRollout) => promotionRollout?.metadata?.name || ''}
        loading={listQuery.isLoading}
        expandable={{
          expandedRowRender: (promotionRollout) => (
            <TargetsTable
              promotionRollout={promotionRollout}
              onSelectPromotion={setSelectedPromotion}
            />
          ),
          rowExpandable: (promotionRollout) => !!promotionRollout?.spec?.targets?.length,
          // The rollout currently fanning out is the one a user came to see.
          defaultExpandedRowKeys: promotionRollouts
            .filter(
              (promotionRollout) =>
                !isPromotionRolloutPhaseTerminal(promotionRollout?.status?.phase) &&
                promotionRollout?.spec?.targets?.length
            )
            .map((promotionRollout) => promotionRollout?.metadata?.name || '')
        }}
      />

      {selectedPromotion && (
        <PromotionComponent
          visible={!!selectedPromotion}
          hide={() => setSelectedPromotion(undefined)}
          promotionId={selectedPromotion}
          project={routeProjectName || projectName}
        />
      )}
    </div>
  );
};
