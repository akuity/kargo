import { faCircleExclamation } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Alert, Flex, Table, Tag, Tooltip, Typography } from 'antd';
import { ColumnsType } from 'antd/es/table';
import { format } from 'date-fns';

import { getShortFreightLabel } from '@ui/features/common/utils';
import { useListPromotionRequests } from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';
import { parseDate } from '@ui/utils/dates';

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
      width: 220,
      render: (_, promotionRequest) => {
        // status.targets carries per-Target progress and appears as the
        // reconciler acts on each one.
        const progress = promotionRequest?.status?.targets || [];
        if (progress.length) {
          return (
            <Flex gap={4} wrap>
              {progress.map((target) => (
                <Tag key={target.name} color={phaseColor(target.phase)}>
                  {target.name}
                </Tag>
              ))}
            </Flex>
          );
        }

        // Nothing has been acted on yet, so fall back to the Targets the
        // request names. spec.targets is resolved once, when the request is
        // created, so it is always populated even before any progress exists.
        const named = promotionRequest?.spec?.targets || [];
        if (!named.length) {
          return (
            <Tooltip title='The Stage governed no Targets when this request was created'>
              <Typography.Text type='secondary' className='text-xs'>
                no Targets
              </Typography.Text>
            </Tooltip>
          );
        }
        return (
          <Flex gap={4} wrap>
            {named.map((target) => (
              <Tag key={target.name}>{target.name}</Tag>
            ))}
          </Flex>
        );
      }
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
