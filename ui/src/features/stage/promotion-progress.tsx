import { Button, Card, Flex, Progress, Tag, Typography } from 'antd';
import { ReactNode } from 'react';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import { getPromotionPhasePresentation } from '@ui/features/common/promotion-status/promotion-phase';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  PromotionStatusPhase
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import { ExecutionDuration } from './promotion-step-duration';
import { StepStatusIcon } from './promotion-step-status-icon';
import { TaskVarTags } from './promotion-task-var-tags';
import { getUserStepAlias, groupPromotionSteps } from './utils/promotion-step-groups';

export const PromotionProgress = ({
  promotion,
  onCurrentStepClick,
  extra
}: {
  promotion: Promotion;
  onCurrentStepClick: (key: string) => void;
  extra?: ReactNode;
}) => {
  const phase = getPromotionStatusPhase(promotion) || PromotionStatusPhase.PENDING;
  const steps = promotion.spec?.steps ?? [];
  const entries = groupPromotionSteps(steps, promotion.spec?.inflatedTasks);
  const tasks = entries.filter((entry) => entry.type === 'task');

  const doneStepCount = steps.filter((_, index) => {
    const status = getPromotionDirectiveStepStatus(index, promotion.status);

    return (
      status === PromotionDirectiveStepStatus.SUCCESS ||
      status === PromotionDirectiveStepStatus.SKIPPED
    );
  }).length;

  const currentStepNumber = promotion.status?.currentStep ?? 0;
  const currentStep = phase !== PromotionStatusPhase.PENDING ? steps[currentStepNumber] : undefined;
  const currentStepStatus = getPromotionDirectiveStepStatus(currentStepNumber, promotion.status);
  const currentStepAlias = getUserStepAlias(currentStep?.as);

  const currentEntry = entries.find((entry) =>
    entry.type === 'task'
      ? entry.steps.some(({ index }) => index === currentStepNumber)
      : entry.index === currentStepNumber
  );
  const currentTask = currentEntry?.type === 'task' ? currentEntry : undefined;
  const currentEntryKey =
    currentEntry?.type === 'task'
      ? currentEntry.key
      : currentStep?.as || `step-${currentStepNumber}`;

  let position = `Step ${currentStepNumber + 1} of ${steps.length}`;

  if (currentTask) {
    const stepInTask = currentTask.steps.findIndex(({ index }) => index === currentStepNumber);
    position = `Task ${tasks.indexOf(currentTask) + 1} of ${tasks.length} · step ${stepInTask + 1} of ${currentTask.steps.length}`;
  }

  return (
    <Card size='small' className='mb-5'>
      <Flex vertical gap={8}>
        <Flex align='center' gap={8}>
          {extra}
          <Tag color={getPromotionPhasePresentation(phase).tagColor}>{phase}</Tag>
          <Typography.Text strong>
            {doneStepCount} / {steps.length} steps done
          </Typography.Text>
          <ExecutionDuration
            meta={{
              startedAt: promotion.metadata?.creationTimestamp,
              finishedAt: promotion.status?.finishedAt
            }}
            progressing={!isPromotionPhaseTerminal(phase)}
          />
        </Flex>

        <Progress
          percent={steps.length ? Math.round((doneStepCount / steps.length) * 100) : 0}
          showInfo={false}
        />

        {currentStep && (
          <Flex align='center' gap={8} wrap>
            <StepStatusIcon status={currentStepStatus} />
            <Button
              type='link'
              size='small'
              className='!p-0'
              onClick={() => onCurrentStepClick(currentEntryKey)}
            >
              <Typography.Text strong code>
                {currentStep.uses}
              </Typography.Text>
            </Button>
            {currentStepAlias && (
              <Tag color='blue' className='text-xs'>
                {currentStepAlias}
              </Tag>
            )}
            {currentTask && (
              <Typography.Text strong>
                in {currentTask.inflatedTask.task?.name ?? currentTask.key}
              </Typography.Text>
            )}
            <Typography.Text type='secondary' className='text-xs'>
              {position}
            </Typography.Text>
            <TaskVarTags vars={currentTask?.inflatedTask.vars ?? []} />
          </Flex>
        )}
      </Flex>
    </Card>
  );
};
