import { faLayerGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tag, Typography } from 'antd';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import { ExecutionDuration } from './promotion-step-duration';
import { StepStatusIcon } from './promotion-step-status-icon';
import { StepTitle } from './promotion-step-title';
import {
  countStepStatuses,
  getTaskGroupExecutionMetadata,
  getTaskGroupStatus,
  getTaskGroupStepStatuses,
  isProgressingStatus,
  PromotionTaskGroup
} from './utils/promotion-step-groups';

const stepStatusLabels: [PromotionDirectiveStepStatus, string][] = [
  [PromotionDirectiveStepStatus.SUCCESS, 'succeeded'],
  [PromotionDirectiveStepStatus.FAILED, 'failed'],
  [PromotionDirectiveStepStatus.RETRYING, 'retrying'],
  [PromotionDirectiveStepStatus.RUNNING, 'running'],
  [PromotionDirectiveStepStatus.SKIPPED, 'skipped']
];

const formatStepCounts = (statuses: PromotionDirectiveStepStatus[]) => {
  const counts = countStepStatuses(statuses);
  const countLabels = stepStatusLabels
    .filter(([status]) => counts[status])
    .map(([status, label]) => `${counts[status]} ${label}`);

  return [`${statuses.length} steps`, ...countLabels].join(' · ');
};

export const PromotionTaskGroupHeader = ({
  group,
  promotion
}: {
  group: PromotionTaskGroup;
  promotion: Promotion;
}) => {
  const statuses = getTaskGroupStepStatuses(group, promotion.status);
  const status = getTaskGroupStatus(statuses);
  const vars = group.task.vars ?? [];

  return (
    <Flex align='start' className='min-w-0'>
      <StepStatusIcon status={status} />
      <Flex vertical gap={4} className='flex-1 min-w-0'>
        <Flex align='center' gap={8}>
          <StepTitle title={group.task.name} />
          <Tag
            color='purple'
            variant='filled'
            icon={<FontAwesomeIcon icon={faLayerGroup} />}
            className='text-xs'
          >
            {group.task.kind}
          </Tag>
        </Flex>
        {!!group.task.description && (
          <Typography.Text
            type='secondary'
            className='text-xs'
            ellipsis
            title={group.task.description}
          >
            {group.task.description}
          </Typography.Text>
        )}
        {vars.length > 0 && (
          <Flex wrap gap={4}>
            {vars.map((variable) => (
              <Tag key={variable.name} className='text-xs'>
                {variable.name}: {variable.value}
              </Tag>
            ))}
          </Flex>
        )}
      </Flex>
      <Flex align='center' gap={8}>
        <Typography.Text type='secondary' className='text-xs'>
          {formatStepCounts(statuses)}
        </Typography.Text>
        <ExecutionDuration
          meta={getTaskGroupExecutionMetadata(group, promotion.status)}
          progressing={isProgressingStatus(status)}
          promotionFinishedAt={promotion.status?.finishedAt}
        />
      </Flex>
    </Flex>
  );
};
