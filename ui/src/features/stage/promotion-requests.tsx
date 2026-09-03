import { faBullseye, faCircleExclamation } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Flex, Table, Tag, Tooltip, Typography } from 'antd';
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
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest, PromotionRequestSummary } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { Promotion as PromotionComponent } from '../project/pipelines/promotion/promotion';

import { useGetFreightMap } from './tabs/freight-history/use-get-freight-map';
import { useWatchPromotionRequests } from './use-watch-promotion-requests';
import {
  PromotionRequestTargetRow,
  blockingMessage,
  isPromotionRequestPhaseTerminal,
  promotionRequestCompareFn,
  targetRows
} from './utils/promotion-request';

// summaryKey maps a phase to its counter in status.summary, whose fields are
// the lower-cased phase names.
const summaryKey = (phase: string) => phase.toLowerCase() as keyof PromotionRequestSummary;

// TargetSummary compresses status.summary into one chip per non-zero phase. A
// request the controller has not yet acted on has no summary, in which case
// every Target it names is, in effect, pending.
const TargetSummary = ({ promotionRequest }: { promotionRequest: PromotionRequest }) => {
  const targetCount = promotionRequest?.spec?.targets?.length || 0;
  if (!targetCount) {
    return (
      <Tooltip title='The Stage governed no Targets when this request was created'>
        <Typography.Text type='secondary' className='text-xs'>
          no Targets
        </Typography.Text>
      </Tooltip>
    );
  }

  const summary = promotionRequest?.status?.summary;

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
  promotionRequest,
  onSelectPromotion
}: {
  promotionRequest: PromotionRequest;
  onSelectPromotion: (name: string) => void;
}) => {
  const columns: ColumnsType<PromotionRequestTargetRow> = [
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
      dataSource={targetRows(promotionRequest)}
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
 * PromotionRequests lists the PromotionRequests belonging to a target-aware
 * Stage, newest first, each expandable to the Targets it names and the child
 * Promotion promoting Freight to each.
 *
 * A target-aware Stage promotes Freight to the Targets it governs by way of a
 * PromotionRequest rather than a Promotion, so for such a Stage the Promotions
 * table alone would show nothing at all -- and a user who just dragged Freight
 * onto the Stage would be left with no indication of what became of it. This
 * surfaces the request and, when it is not going to progress, the reason.
 */
export const PromotionRequests = ({ projectName, stageName }: Props) => {
  const { name: routeProjectName } = useParams();

  const listQuery = useListPromotionRequests(
    projectName,
    { stage: stageName },
    { query: { enabled: !!projectName && !!stageName } }
  );

  useWatchPromotionRequests(projectName, stageName, !listQuery.isLoading);

  const freightMap = useGetFreightMap(routeProjectName || projectName);

  const [selectedPromotion, setSelectedPromotion] = useState<string | undefined>();

  const promotionRequests = useMemo(
    () => [...(listQuery.data?.data?.items || [])].sort(promotionRequestCompareFn),
    [listQuery.data]
  );

  if (!promotionRequests.length) {
    return null;
  }

  const blocked = blockingMessage(promotionRequests[0]);

  const columns: ColumnsType<PromotionRequest> = [
    {
      title: 'Phase',
      width: 110,
      render: (_, promotionRequest) => {
        const phase = promotionRequest?.status?.phase;
        return (
          <Tag color={getPromotionPhasePresentation(phase).tagColor}>{phase || 'Pending'}</Tag>
        );
      }
    },
    {
      title: 'Name',
      render: (_, promotionRequest) => (
        <Typography.Text copyable className='text-xs font-mono'>
          {promotionRequest?.metadata?.name}
        </Typography.Text>
      )
    },
    {
      title: 'Freight',
      width: 160,
      render: (_, promotionRequest) => {
        const freightName = promotionRequest?.spec?.freight || '';
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
      render: (_, promotionRequest) => <TargetSummary promotionRequest={promotionRequest} />
    },
    {
      title: 'Created',
      width: 180,
      render: (_, promotionRequest) => {
        const date = parseDate(promotionRequest?.metadata?.creationTimestamp);
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
      {blocked && (
        <Alert
          className='mb-3'
          type='warning'
          showIcon
          icon={<FontAwesomeIcon icon={faCircleExclamation} />}
          message='This Stage promotes to Targets'
          description={blocked}
        />
      )}
      <Typography.Title level={5} className='mb-2'>
        Promotion Requests
      </Typography.Title>
      <Table
        dataSource={promotionRequests}
        columns={columns}
        size='small'
        pagination={{ hideOnSinglePage: true, pageSize: 5 }}
        rowKey={(promotionRequest) => promotionRequest?.metadata?.name || ''}
        loading={listQuery.isLoading}
        expandable={{
          expandedRowRender: (promotionRequest) => (
            <TargetsTable
              promotionRequest={promotionRequest}
              onSelectPromotion={setSelectedPromotion}
            />
          ),
          rowExpandable: (promotionRequest) => !!promotionRequest?.spec?.targets?.length,
          // The request currently fanning out is the one a user came to see.
          defaultExpandedRowKeys: promotionRequests
            .filter(
              (promotionRequest) =>
                !isPromotionRequestPhaseTerminal(promotionRequest?.status?.phase) &&
                promotionRequest?.spec?.targets?.length
            )
            .map((promotionRequest) => promotionRequest?.metadata?.name || '')
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
