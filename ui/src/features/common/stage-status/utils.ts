import { Tone } from '@ui/features/common/status-tone';
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

export type StagePhase = 'Promoting' | 'Verifying' | 'Reconciling' | 'Failed' | 'Ready' | 'Unknown';

// The tone each phase is stated in, wherever it is stated. Promoting and
// Verifying are both work under way; Unknown is not a verdict and steps back.
export const phaseTone: Record<StagePhase, Tone> = {
  Promoting: 'processing',
  Verifying: 'processing',
  Reconciling: 'warning',
  Failed: 'error',
  Ready: 'success',
  Unknown: 'muted'
};

// getStageStatus walks a Stage's conditions once, in priority order, and
// reports where the Stage stands along with whatever the deciding condition
// had to say for itself. Everything that states a Stage's status -- the word,
// the mark beside it, the tone both are drawn in -- comes from here, so none
// of them can drift away from the others.
//
// NoFreight stands a condition down rather than asserting anything, so a
// condition carrying that reason is passed over.
export const getStageStatus = (
  stage: Stage,
  isControllerDead?: boolean
): { phase: StagePhase; message: string } => {
  // A dead (or absent) controller means any condition the Stage carries was
  // written before the controller stopped reporting and is now stale. Surface
  // this as Failed regardless of what those stale conditions say. The reason
  // is the controller's rather than the Stage's, so it is left to the caller,
  // which is the only one that knows which controller went silent.
  if (isControllerDead) {
    return { phase: 'Failed', message: '' };
  }

  const conditions = stage?.status?.conditions || [];

  const promoting = hasCondition(
    StageConditionType.Promoting,
    StageConditionStatus.True,
    conditions
  );

  if (promoting.isActive && promoting.condition?.reason !== 'NoFreight') {
    return { phase: 'Promoting', message: promoting.condition?.message ?? '' };
  }

  const verifying = hasCondition(
    StageConditionType.Verified,
    StageConditionStatus.Unknown,
    conditions
  );

  if (verifying.isActive && verifying.condition?.reason !== 'NoFreight') {
    return { phase: 'Verifying', message: verifying.condition?.message ?? '' };
  }

  const reconciling = hasCondition(
    StageConditionType.Reconciling,
    StageConditionStatus.True,
    conditions
  );

  if (reconciling.isActive) {
    return { phase: 'Reconciling', message: reconciling.condition?.message ?? '' };
  }

  const ready = hasCondition(StageConditionType.Ready, StageConditionStatus.True, conditions);

  const failed = ready.condition?.status === StageConditionStatus.False;

  if (failed && ready.condition?.reason !== 'NoFreight') {
    return { phase: 'Failed', message: ready.condition?.message ?? '' };
  }

  if (ready.isActive) {
    return { phase: 'Ready', message: ready.condition?.message ?? '' };
  }

  return { phase: 'Unknown', message: '' };
};

export const getStagePhase = (stage: Stage, isControllerDead?: boolean): StagePhase =>
  getStageStatus(stage, isControllerDead).phase;
