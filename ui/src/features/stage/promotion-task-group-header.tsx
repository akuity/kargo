import { faLayerGroup } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex, Tag, Typography } from 'antd';

import { PromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import { ExecutionDuration } from './promotion-step-duration';
import { StepStatusIcon } from './promotion-step-status-icon';
import { StepTitle } from './promotion-step-title';
import { TaskVarTags } from './promotion-task-var-tags';
import {
  countStepStatuses,
  didPromotionEndUnsuccessfullyInTaskGroup,
  getEffectiveVars,
  getTaskGroupExecutionMetadata,
  getTaskGroupStatus,
  getTaskGroupStepStatuses,
  isProgressingStatus,
  PromotionTaskGroup,
  stepStatusLabels
} from './utils/promotion-step-groups';

const formatStepCounts = (statuses: PromotionDirectiveStepStatus[]) => {
  const counts = countStepStatuses(statuses);
  const countLabels = stepStatusLabels
    .filter(([status]) => counts[status])
    .map(([status, label]) => `${counts[status]} ${label}`);

  return [`${statuses.length} step${statuses.length === 1 ? '' : 's'}`, ...countLabels].join(' · ');
};

export const PromotionTaskGroupHeader = ({
  group,
  promotion
}: {
  group: PromotionTaskGroup;
  promotion: Promotion;
}) => {
  const statuses = getTaskGroupStepStatuses(group, promotion.status);
  const status = getTaskGroupStatus(
    statuses,
    didPromotionEndUnsuccessfullyInTaskGroup(group, promotion.status)
  );
  const { inflatedTask } = group;
  const vars = getEffectiveVars(inflatedTask.vars);

  return (
    <Flex align='start' className='min-w-0'>
      <StepStatusIcon status={status} />
      <Flex vertical gap={4} className='flex-1 min-w-0'>
        <Flex align='center' gap={8}>
          <StepTitle title={inflatedTask.task?.name ?? group.key} />
          <Tag
            color='purple'
            variant='filled'
            icon={<FontAwesomeIcon icon={faLayerGroup} />}
            className='text-xs'
          >
            {inflatedTask.task?.kind}
          </Tag>
        </Flex>
        {!!inflatedTask.description && (
          <Typography.Text type='secondary' className='text-xs'>
            {inflatedTask.description}
          </Typography.Text>
        )}
        {vars.length > 0 && <TaskVarTags vars={vars} />}
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
