import { Button, Card, Flex, Progress, Tag, Typography } from 'antd';
import { ReactNode } from 'react';

import { getPromotionDirectiveStepStatus } from '@ui/features/common/promotion-directive-step-status/utils';
import { isPromotionPhaseTerminal } from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import { ExecutionDuration } from './promotion-step-duration';
import { StepStatusIcon } from './promotion-step-status-icon';
import { TaskVarTags } from './promotion-task-var-tags';
import { useGetPromotionProgress } from './use-get-promotion-progress';
import { getStepKey, getTaskGroupName, getUserStepAlias } from './utils/promotion-step-groups';

export const PromotionProgress = ({
  promotion,
  onCurrentStepClick,
  extra
}: {
  promotion: Promotion;
  onCurrentStepClick?: (key: string) => void;
  extra?: ReactNode;
}) => {
  const { phase, tasks, currentStep, currentStepIndex, currentTask, stepCount, doneStepCount } =
    useGetPromotionProgress(promotion);
  const currentAlias = getUserStepAlias(currentStep?.as);

  return (
    <Card size='small' className='mb-5'>
      <Flex vertical gap={8}>
        <Flex align='center' gap={8}>
          {extra}
          <Typography.Text strong>
            {doneStepCount} / {stepCount} steps done
          </Typography.Text>
          <ExecutionDuration
            meta={{
              startedAt: promotion.status?.stepExecutionMetadata?.[0]?.startedAt,
              finishedAt: promotion.status?.finishedAt
            }}
            progressing={!isPromotionPhaseTerminal(phase)}
          />
        </Flex>

        <Progress
          percent={stepCount ? Math.round((doneStepCount / stepCount) * 100) : 0}
          showInfo={false}
        />

        {currentStep && (
          <Flex align='center' gap={8} wrap>
            <StepStatusIcon
              status={getPromotionDirectiveStepStatus(currentStepIndex, promotion.status)}
            />
            <Button
              type='link'
              size='small'
              className='!p-0'
              disabled={!onCurrentStepClick}
              onClick={() =>
                onCurrentStepClick?.(currentTask?.key ?? getStepKey(currentStep, currentStepIndex))
              }
            >
              <Typography.Text strong code>
                {currentStep.uses}
              </Typography.Text>
            </Button>
            {currentAlias && (
              <Tag color='blue' className='text-xs'>
                {currentAlias}
              </Tag>
            )}
            {currentTask && (
              <Typography.Text strong>in {getTaskGroupName(currentTask)}</Typography.Text>
            )}
            <Typography.Text type='secondary' className='text-xs'>
              {currentTask
                ? `Task ${tasks.indexOf(currentTask) + 1} of ${tasks.length} · step ${currentTask.steps.findIndex(({ index }) => index === currentStepIndex) + 1} of ${currentTask.steps.length}`
                : `Step ${currentStepIndex + 1} of ${stepCount}`}
            </Typography.Text>
            <TaskVarTags vars={currentTask?.inflatedTask.vars ?? []} />
          </Flex>
        )}
      </Flex>
    </Card>
  );
};
