import { useMemo } from 'react';

import {
  getPromotionDirectiveStepStatus,
  PromotionDirectiveStepStatus
} from '@ui/features/common/promotion-directive-step-status/utils';
import {
  getPromotionStatusPhase,
  PromotionStatusPhase
} from '@ui/features/common/promotion-status/utils';
import { Promotion } from '@ui/gen/api/v2/models';

import {
  countStepStatuses,
  groupPromotionSteps,
  PromotionTaskGroup
} from './utils/promotion-step-groups';

export const useGetPromotionProgress = (promotion: Promotion) =>
  useMemo(() => {
    const phase = getPromotionStatusPhase(promotion) || PromotionStatusPhase.PENDING;
    const steps = promotion.spec?.steps ?? [];
    const entries = groupPromotionSteps(steps, promotion.spec?.inflatedTasks);
    const tasks = entries.filter((entry): entry is PromotionTaskGroup => entry.type === 'task');
    const counts = countStepStatuses(
      steps.map((_, index) => getPromotionDirectiveStepStatus(index, promotion.status))
    );
    const currentStepIndex = promotion.status?.currentStep ?? 0;

    return {
      phase,
      tasks,
      currentStepIndex,
      currentStep: phase === PromotionStatusPhase.PENDING ? undefined : steps[currentStepIndex],
      stepCount: steps.length,
      doneStepCount:
        (counts[PromotionDirectiveStepStatus.SUCCESS] ?? 0) +
        (counts[PromotionDirectiveStepStatus.SKIPPED] ?? 0),
      currentTask: tasks.find((task) => task.steps.some(({ index }) => index === currentStepIndex))
    };
  }, [promotion]);
