import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';
import { Condition } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

export const enum StageConditionType {
  Promoting = 'Promoting',
  Reconciling = 'Reconciling',
  Ready = 'Ready',
  Verified = 'Verified'
}

export const enum StageConditionStatus {
  True = 'True',
  False = 'False',
  Unknown = 'Unknown'
}

export const hasCondition = (
  type: StageConditionType,
  status: StageConditionStatus,
  conditions: Condition[]
): { condition: Condition | undefined; isActive: boolean } => {
  const condition = conditions.find((c) => c.type === type);
  return {
    condition,
    isActive: condition?.status === status
  };
};

export const getStagePhase = (stage: Stage) => {
  const conditions = stage?.status?.conditions || [];

  const promoting = hasCondition(
    StageConditionType.Promoting,
    StageConditionStatus.True,
    conditions
  );

  if (promoting.isActive) {
    return 'Promoting';
  }

  const verifying = hasCondition(
    StageConditionType.Verified,
    StageConditionStatus.Unknown,
    conditions
  );

  if (verifying.isActive) {
    return 'Verifying';
  }

  const ready = hasCondition(StageConditionType.Reconciling, StageConditionStatus.True, conditions);

  const failed = ready.condition?.status === StageConditionStatus.False;

  if (failed) {
    return 'Failed';
  }

  if (ready.isActive) {
    return 'Ready';
  }
};
