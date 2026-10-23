import { Button, Card, Flex, Tag, Tooltip, Typography, theme } from 'antd';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import {
  getPromotionPhasePresentation,
  PhaseColorToken
} from '@ui/features/common/promotion-status/promotion-phase';
import {
  getPromotionStatusPhase,
  isPromotionPhaseTerminal,
  PromotionStatusPhase
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import { ExecutionDuration } from './promotion-step-duration';
import { StepStatusIcon } from './promotion-step-status-icon';
import { TaskVarTags } from './promotion-task-var-tags';
import {
  countStepStatuses,
  getTaskGroupStatus,
  getTaskGroupStepStatuses,
  getUserStepAlias,
  groupPromotionSteps,
  stepStatusLabels
} from './utils/promotion-step-groups';

const maxStepsDrawnAsTicks = 200;

const stepStatusColorTokens: Record<
  PromotionDirectiveStepStatus,
  PhaseColorToken | 'colorWarning' | 'colorBgContainer' | 'colorFill'
> = {
  [PromotionDirectiveStepStatus.SUCCESS]: 'colorSuccess',
  [PromotionDirectiveStepStatus.RUNNING]: 'colorInfo',
  [PromotionDirectiveStepStatus.RETRYING]: 'colorWarning',
  [PromotionDirectiveStepStatus.FAILED]: 'colorError',
  [PromotionDirectiveStepStatus.SKIPPED]: 'colorBgContainer',
  [PromotionDirectiveStepStatus.WONT_RUN]: 'colorFill'
};

export const PromotionProgress = ({
  promotion,
  onSegmentClick
}: {
  promotion: Promotion;
  onSegmentClick: (key: string) => void;
}) => {
  const { token } = theme.useToken();

  const phase = getPromotionStatusPhase(promotion) || PromotionStatusPhase.PENDING;
  const steps = promotion.spec?.steps ?? [];
  const drawStepTicks = steps.length <= maxStepsDrawnAsTicks;

  const segments = groupPromotionSteps(steps).map((entry) => {
    if (entry.type === 'task') {
      const statuses = getTaskGroupStepStatuses(entry, promotion.status);

      return {
        key: entry.key,
        isTask: true,
        title: entry.task.name,
        vars: entry.task.vars ?? [],
        steps: entry.steps,
        statuses,
        status: getTaskGroupStatus(statuses)
      };
    }

    const status = getPromotionDirectiveStepStatus(entry.index, promotion.status);

    return {
      key: entry.step.as || `step-${entry.index}`,
      isTask: false,
      title: entry.step.uses,
      vars: [],
      steps: [entry],
      statuses: [status],
      status
    };
  });

  const tasks = segments.filter((segment) => segment.isTask);

  const hasCurrentStep =
    phase !== PromotionStatusPhase.PENDING && phase !== PromotionStatusPhase.SUCCEEDED;
  const currentStepNumber = promotion.status?.currentStep ?? 0;

  const current = hasCurrentStep
    ? segments.find((segment) => segment.steps.some(({ index }) => index === currentStepNumber))
    : undefined;
  const currentStepIndex =
    current?.steps.findIndex(({ index }) => index === currentStepNumber) ?? -1;
  const currentStep = current?.steps[currentStepIndex];
  const currentStepAlias = getUserStepAlias(currentStep?.step.as);

  let position = '';

  if (current?.isTask) {
    position = `Task ${tasks.indexOf(current) + 1} of ${tasks.length} · step ${currentStepIndex + 1} of ${current.steps.length}`;
  } else if (currentStep) {
    position = `Step ${currentStep.index + 1} of ${steps.length}`;
  }

  const doneStepCount = segments
    .flatMap((segment) => segment.statuses)
    .filter(
      (status) =>
        status === PromotionDirectiveStepStatus.SUCCESS ||
        status === PromotionDirectiveStepStatus.SKIPPED
    ).length;

  const taskCounts = countStepStatuses(tasks.map((task) => task.status));

  const tally = [
    tasks.length > 0 && `${tasks.length} tasks`,
    ...stepStatusLabels
      .filter(([status]) => taskCounts[status])
      .map(([status, label]) => `${taskCounts[status]} ${label}`)
  ]
    .filter(Boolean)
    .join(' · ');

  return (
    <Card size='small' className='mb-5'>
      <Flex vertical gap={12}>
        <Flex align='center' gap={8}>
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

        <Flex gap={2}>
          {segments.map((segment) => {
            const ticks = drawStepTicks ? segment.statuses : [segment.status];

            return (
              <Tooltip
                key={segment.key}
                title={[
                  segment.title,
                  ...segment.vars.map((variable) => `${variable.name}: ${variable.value}`)
                ].join(' · ')}
              >
                <Button
                  type='text'
                  className='!h-auto !px-0 !py-1.5 !border-0 min-w-0'
                  style={{
                    flex: `${ticks.length} 1 0`,
                    background: segment === current ? token.controlItemBgActive : undefined
                  }}
                  onClick={() => onSegmentClick(segment.key)}
                >
                  <Flex gap={2} align='center' className='w-full h-3'>
                    {ticks.map((status, i) => {
                      const isCurrentStep =
                        segment === current && (!drawStepTicks || i === currentStepIndex);

                      return (
                        <div
                          key={i}
                          className={`flex-1 rounded-sm ${isCurrentStep ? 'h-3' : 'h-2'}`}
                          style={{
                            background: token[stepStatusColorTokens[status]],
                            boxShadow:
                              status === PromotionDirectiveStepStatus.SKIPPED
                                ? `inset 0 0 0 1px ${token.colorTextQuaternary}`
                                : undefined,
                            outline: isCurrentStep
                              ? `1px solid ${token[stepStatusColorTokens[status]]}`
                              : undefined,
                            outlineOffset: 1
                          }}
                        />
                      );
                    })}
                  </Flex>
                </Button>
              </Tooltip>
            );
          })}
        </Flex>

        <Flex align='center' gap={8} wrap>
          {current && currentStep && (
            <>
              <StepStatusIcon status={current.statuses[currentStepIndex]} />
              <Typography.Text strong code>
                {currentStep.step.uses}
              </Typography.Text>
              {currentStepAlias && (
                <Tag color='blue' className='text-xs'>
                  {currentStepAlias}
                </Tag>
              )}
              {current.isTask && <Typography.Text strong>in {current.title}</Typography.Text>}
              <Typography.Text type='secondary' className='text-xs'>
                {position}
              </Typography.Text>
              <TaskVarTags vars={current.vars} />
            </>
          )}
          <Typography.Text type='secondary' className='ml-auto text-xs'>
            {tally}
          </Typography.Text>
        </Flex>
      </Flex>
    </Card>
  );
};
