import {
  ExpressionVariable,
  InflatedPromotionTask,
  PromotionStatus,
  PromotionStep,
  StepExecutionMetadata
} from '@ui/gen/api/v2/models';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '../../common/promotion-directive-step-status/utils';
import {
  isPromotionPhaseTerminal,
  PromotionStatusPhase
} from '../../common/promotion-status/utils';

const taskStepAliasSeparator = '::';

const generatedStepAliasRegex = /^(step|task)-\d+$/;

export const stepStatusLabels: [PromotionDirectiveStepStatus, string][] = [
  [PromotionDirectiveStepStatus.SUCCESS, 'succeeded'],
  [PromotionDirectiveStepStatus.FAILED, 'failed'],
  [PromotionDirectiveStepStatus.RETRYING, 'retrying'],
  [PromotionDirectiveStepStatus.RUNNING, 'running'],
  [PromotionDirectiveStepStatus.SKIPPED, 'skipped']
];

export type IndexedPromotionStep = {
  type: 'step';
  step: PromotionStep;
  index: number;
};

export type PromotionTaskGroup = {
  type: 'task';
  key: string;
  inflatedTask: InflatedPromotionTask;
  steps: IndexedPromotionStep[];
};

export type PromotionStepEntry = IndexedPromotionStep | PromotionTaskGroup;

export const getUserStepAlias = (alias?: string) => {
  const userAlias = alias?.split(taskStepAliasSeparator).at(-1);

  if (!userAlias || generatedStepAliasRegex.test(userAlias)) {
    return undefined;
  }

  return userAlias;
};

export const getStepKey = (step: PromotionStep, index: number) => step.as || `step-${index}`;

export const getTaskGroupName = (group: PromotionTaskGroup) =>
  group.inflatedTask.task?.name ?? group.key;

const getInflatedTasksByStepAlias = (inflatedTasks: InflatedPromotionTask[]) =>
  new Map(
    inflatedTasks.flatMap((inflatedTask) =>
      (inflatedTask.steps ?? []).map((stepAlias) => [stepAlias, inflatedTask] as const)
    )
  );

export const groupPromotionSteps = (
  steps: PromotionStep[],
  inflatedTasks: InflatedPromotionTask[] = []
): PromotionStepEntry[] => {
  const inflatedTasksByStepAlias = getInflatedTasksByStepAlias(inflatedTasks);
  const entries: PromotionStepEntry[] = [];

  steps.forEach((step, index) => {
    const indexedStep: IndexedPromotionStep = { type: 'step', step, index };
    const inflatedTask = step.as ? inflatedTasksByStepAlias.get(step.as) : undefined;

    if (!inflatedTask) {
      entries.push(indexedStep);
      return;
    }

    const previousEntry = entries.at(-1);

    if (previousEntry?.type === 'task' && previousEntry.inflatedTask.alias === inflatedTask.alias) {
      previousEntry.steps.push(indexedStep);
      return;
    }

    entries.push({
      type: 'task',
      key: inflatedTask.alias || `task-${index}`,
      inflatedTask,
      steps: [indexedStep]
    });
  });

  return entries;
};

export const getTaskGroupStepStatuses = (
  group: PromotionTaskGroup,
  promotionStatus?: PromotionStatus
) => group.steps.map(({ index }) => getPromotionDirectiveStepStatus(index, promotionStatus));

export const didPromotionEndUnsuccessfullyInTaskGroup = (
  group: PromotionTaskGroup,
  promotionStatus?: PromotionStatus
) => {
  const phase = promotionStatus?.phase as PromotionStatusPhase;

  if (!isPromotionPhaseTerminal(phase) || phase === PromotionStatusPhase.SUCCEEDED) {
    return false;
  }

  const currentStep = promotionStatus?.currentStep ?? 0;

  return group.steps.some(({ index }) => index === currentStep);
};

export const getTaskGroupStatus = (
  statuses: PromotionDirectiveStepStatus[],
  promotionEndedUnsuccessfully = false
): PromotionDirectiveStepStatus => {
  if (promotionEndedUnsuccessfully) {
    return PromotionDirectiveStepStatus.FAILED;
  }

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

export const getEffectiveVars = (vars: ExpressionVariable[] = []) => [
  ...new Map(vars.map((variable) => [variable.name, variable])).values()
];
