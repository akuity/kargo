import {
  PromotionStatus,
  PromotionStep,
  PromotionStepTaskRef,
  StepExecutionMetadata
} from '@ui/gen/api/v2/models';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '../../common/promotion-directive-step-status/utils';

const taskStepAliasSeparator = '::';

const generatedStepAliasRegex = /^(step|task)-\d+$/;

export type IndexedPromotionStep = {
  type: 'step';
  step: PromotionStep;
  index: number;
};

export type PromotionTaskGroup = {
  type: 'task';
  key: string;
  task: PromotionStepTaskRef;
  steps: IndexedPromotionStep[];
};

export type PromotionStepEntry = IndexedPromotionStep | PromotionTaskGroup;

const getTaskAlias = (step: PromotionStep) => {
  if (!step.ref || !step.as?.includes(taskStepAliasSeparator)) {
    return undefined;
  }

  return step.as.split(taskStepAliasSeparator)[0];
};

export const getUserStepAlias = (alias?: string) => {
  const userAlias = alias?.split(taskStepAliasSeparator).at(-1);

  if (!userAlias || generatedStepAliasRegex.test(userAlias)) {
    return undefined;
  }

  return userAlias;
};

export const groupPromotionSteps = (steps: PromotionStep[]): PromotionStepEntry[] => {
  const entries: PromotionStepEntry[] = [];

  steps.forEach((step, index) => {
    const indexedStep: IndexedPromotionStep = { type: 'step', step, index };

    const sourceTask = step.ref;
    const taskAlias = getTaskAlias(step);
    const isStepInflatedFromTask = !!sourceTask && !!taskAlias;

    const previousEntry = entries.at(-1);
    const previousTaskGroup = previousEntry?.type === 'task' ? previousEntry : undefined;

    if (isStepInflatedFromTask) {
      const continuesPreviousTaskGroup = previousTaskGroup?.key === taskAlias;

      if (previousTaskGroup && continuesPreviousTaskGroup) {
        previousTaskGroup.steps.push(indexedStep);
      } else {
        entries.push({ type: 'task', key: taskAlias, task: sourceTask, steps: [indexedStep] });
      }
    } else {
      entries.push(indexedStep);
    }
  });

  return entries;
};

export const getTaskGroupStepStatuses = (
  group: PromotionTaskGroup,
  promotionStatus?: PromotionStatus
) => group.steps.map(({ index }) => getPromotionDirectiveStepStatus(index, promotionStatus));

export const getTaskGroupStatus = (
  statuses: PromotionDirectiveStepStatus[]
): PromotionDirectiveStepStatus => {
  const has = (status: PromotionDirectiveStepStatus) => statuses.includes(status);
  const all = (status: PromotionDirectiveStepStatus) => statuses.every((s) => s === status);

  if (has(PromotionDirectiveStepStatus.FAILED)) {
    return PromotionDirectiveStepStatus.FAILED;
  }

  if (has(PromotionDirectiveStepStatus.RETRYING)) {
    return PromotionDirectiveStepStatus.RETRYING;
  }

  if (has(PromotionDirectiveStepStatus.RUNNING)) {
    return PromotionDirectiveStepStatus.RUNNING;
  }

  if (all(PromotionDirectiveStepStatus.SKIPPED)) {
    return PromotionDirectiveStepStatus.SKIPPED;
  }

  if (all(PromotionDirectiveStepStatus.WONT_RUN)) {
    return PromotionDirectiveStepStatus.WONT_RUN;
  }

  if (has(PromotionDirectiveStepStatus.WONT_RUN)) {
    return PromotionDirectiveStepStatus.RUNNING;
  }

  return PromotionDirectiveStepStatus.SUCCESS;
};

export const isProgressingStatus = (status: PromotionDirectiveStepStatus) =>
  status === PromotionDirectiveStepStatus.RUNNING ||
  status === PromotionDirectiveStepStatus.RETRYING;

export const countStepStatuses = (statuses: PromotionDirectiveStepStatus[]) =>
  statuses.reduce<Partial<Record<PromotionDirectiveStepStatus, number>>>(
    (counts, status) => ({ ...counts, [status]: (counts[status] ?? 0) + 1 }),
    {}
  );

export const getTaskGroupExecutionMetadata = (
  group: PromotionTaskGroup,
  promotionStatus?: PromotionStatus
): StepExecutionMetadata => {
  const metas = group.steps
    .map(({ index }) => promotionStatus?.stepExecutionMetadata?.[index])
    .filter((meta) => !!meta);

  return {
    startedAt: metas.find((meta) => meta.startedAt)?.startedAt,
    finishedAt: metas.filter((meta) => meta.finishedAt).at(-1)?.finishedAt
  };
};
