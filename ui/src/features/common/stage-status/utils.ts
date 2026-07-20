import { Stage, V1Condition } from '@ui/gen/api/v2/models';

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

// Ready=False reasons that describe a transient, self-resolving state (e.g. an
// Argo CD Application still rolling out after a successful promotion) rather
// than an actual failure. The controller copies the Healthy condition's reason
// onto the Ready condition, so these mirror the non-terminal health states.
const transientNotReadyReasons = ['Progressing', 'WaitingForHealthCheck'];

export const isTransientNotReadyReason = (reason?: string) =>
  !!reason && transientNotReadyReasons.includes(reason);

export const hasCondition = (
  type: StageConditionType,
  status: StageConditionStatus,
  conditions: V1Condition[]
): { condition: V1Condition | undefined; isActive: boolean } => {
  const condition = conditions.find((c) => c.type === type);
  return {
    condition,
    isActive: condition?.status === status
  };
};

export const getStagePhase = (stage: Stage, isControllerDead?: boolean) => {
  // A dead (or absent) controller means any condition the Stage carries
  // was written before the controller stopped reporting and is now stale.
  // Surface this as Failed regardless of what those stale conditions say.
  if (isControllerDead) {
    return 'Failed';
  }

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

  if (reconciling.isActive) {
    return 'Reconciling';
  }

  const ready = hasCondition(StageConditionType.Ready, StageConditionStatus.True, conditions);

  const notReady = ready.condition?.status === StageConditionStatus.False;

  if (notReady && isTransientNotReadyReason(ready.condition?.reason)) {
    return 'Progressing';
  }

  if (notReady && ready.condition?.reason !== 'NoFreight') {
    return 'Failed';
  }

  if (ready.isActive) {
    return 'Ready';
  }

  return 'Unknown';
};
