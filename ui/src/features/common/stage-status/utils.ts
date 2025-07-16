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

  if (promoting.isActive && promoting.condition?.reason !== 'NoFreight') {
    return 'Promoting';
  }

  const verifying = hasCondition(
    StageConditionType.Verified,
    StageConditionStatus.Unknown,
    conditions
  );

  if (verifying.isActive && verifying.condition?.reason !== 'NoFreight') {
    return 'Verifying';
  }

  const reconciling = hasCondition(
    StageConditionType.Reconciling,
    StageConditionStatus.True,
    conditions
  );

  if (reconciling.isActive && reconciling.condition?.reason !== 'NoFreight') {
    return 'Reconciling';
  }

  const ready = hasCondition(StageConditionType.Ready, StageConditionStatus.True, conditions);

  const failed = ready.condition?.status === StageConditionStatus.False;

  if (failed && ready.condition?.reason !== 'NoFreight') {
    return 'Failed';
  }

  if (ready.isActive && ready.condition?.reason !== 'NoFreight') {
    return 'Ready';
  }

  return 'Unknown';
};
