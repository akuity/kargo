import { faLayerGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Popover, Tag, Typography } from 'antd';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { ExpressionVariable, Promotion } from '@ui/gen/api/v2/models';

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

const maxVisibleVars = 3;

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

const VarTag = ({ variable }: { variable: ExpressionVariable }) => {
  const text = `${variable.name}: ${variable.value}`;

  return (
    <Tag className='text-xs max-w-full truncate' title={text}>
      {text}
    </Tag>
  );
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
  const visibleVars = vars.slice(0, maxVisibleVars);
  const hiddenVars = vars.slice(maxVisibleVars);

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
          <Typography.Text type='secondary' className='text-xs'>
            {group.task.description}
          </Typography.Text>
        )}
        {vars.length > 0 && (
          <Flex wrap gap={4}>
            {visibleVars.map((variable) => (
              <VarTag key={variable.name} variable={variable} />
            ))}
            {hiddenVars.length > 0 && (
              <Popover
                content={
                  <Flex vertical gap={4} align='start' className='max-w-md'>
                    {hiddenVars.map((variable) => (
                      <VarTag key={variable.name} variable={variable} />
                    ))}
                  </Flex>
                }
              >
                <Tag className='text-xs'>+{hiddenVars.length} more</Tag>
              </Popover>
            )}
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
