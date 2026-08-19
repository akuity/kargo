import {
  faCancel,
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Flex, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';

import { getShortFreightLabel } from '@ui/features/common/utils';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

import { useWatchPromotionRequests } from './use-watch-promotion-requests';

const phaseColor = (phase?: string) => {
  switch (phase) {
    case 'Succeeded':
      return 'success';
    case 'Running':
      return 'processing';
    case 'Failed':
    case 'Errored':
      return 'error';
    default:
      return 'default';
  }
};

// readyCondition returns the PromotionRequest's Ready condition, which is where
// a reason for "nothing happened" is reported.
const readyCondition = (promotionRequest: PromotionRequest) =>
  promotionRequest?.status?.conditions?.find((condition) => condition.type === 'Ready');

// blockingMessage returns the message a user needs to see when a
// PromotionRequest is not going to progress -- most commonly that fanning
// Freight out to Targets is not available in this installation.
const blockingMessage = (promotionRequest: PromotionRequest) => {
  const ready = readyCondition(promotionRequest);
  if (ready?.status === 'False' && ready?.message) {
    return ready.message;
  }
  return undefined;
};

const targetPhases = [
  { key: 'succeeded', label: 'succeeded', color: 'success', icon: faCircleCheck },
  { key: 'running', label: 'running', color: 'processing', icon: faCircleNotch, spin: true },
  { key: 'pending', label: 'pending', color: 'default', icon: faHourglassStart },
  { key: 'failed', label: 'failed', color: 'error', icon: faCircleExclamation },
  { key: 'errored', label: 'errored', color: 'error', icon: faCircleExclamation },
  { key: 'aborted', label: 'aborted', color: 'default', icon: faCancel }
] as const;

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
      {targetPhases.map((targetPhase) => {
        const { key, label, color, icon } = targetPhase;
        const count = summary?.[key] ?? (!summary && key === 'pending' ? targetCount : 0);
        if (!count) {
          return null;
        }
        const description = `${count} ${label} Target${count === 1 ? '' : 's'}`;
        return (
          <Tooltip key={key} title={description}>
            <Tag
              className='m-0'
              color={color}
              icon={
                <FontAwesomeIcon icon={icon} spin={'spin' in targetPhase && targetPhase.spin} />
              }
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

type Props = {
  projectName: string;
  stageName: string;
};

/**
 * PromotionRequests lists the PromotionRequests belonging to a Stage.
 *
 * A target-aware Stage promotes Freight to the Targets it governs by way of a
 * PromotionRequest rather than a Promotion, so for such a Stage the Promotions
 * table alone would show nothing at all -- and a user who just dragged Freight
 * onto the Stage would be left with no indication of what became of it. This
 * surfaces the request and, when it is not going to progress, the reason.
 */
export const PromotionRequests = ({ projectName, stageName }: Props) => {
  const listQuery = useListPromotionRequests(
    projectName,
    { stage: stageName },
    { query: { enabled: !!projectName && !!stageName } }
  );

  useWatchPromotionRequests(projectName, stageName, !listQuery.isLoading);

  const promotionRequests = listQuery.data?.data?.items || [];

  if (!promotionRequests.length) {
    return null;
  }

  // Show the newest first. Names embed a ULID, so name order is creation order.
  const sorted = [...promotionRequests].sort((lhs, rhs) =>
    (rhs?.metadata?.name || '').localeCompare(lhs?.metadata?.name || '')
  );

  const blocked = sorted.map(blockingMessage).find(Boolean);

  const columns: ColumnsType<PromotionRequest> = [
    {
      title: 'Phase',
      width: 120,
      render: (_, promotionRequest) => {
        const phase = promotionRequest?.status?.phase;
        return <Tag color={phaseColor(phase)}>{phase || 'Pending'}</Tag>;
      }
    },
    {
      title: 'Name',
      render: (_, promotionRequest) => (
        <Typography.Text copyable className='text-xs'>
          {promotionRequest?.metadata?.name}
        </Typography.Text>
      )
    },
    {
      title: 'Freight',
      width: 160,
      render: (_, promotionRequest) => (
        <Typography.Text className='text-xs'>
          {getShortFreightLabel(promotionRequest?.spec?.freight)}
        </Typography.Text>
      )
    },
    {
      title: 'Targets',
      width: 240,
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
        dataSource={sorted}
        columns={columns}
        size='small'
        pagination={{ hideOnSinglePage: true, pageSize: 5 }}
        rowKey={(promotionRequest) => promotionRequest?.metadata?.name || ''}
        loading={listQuery.isLoading}
      />
    </div>
  );
};
