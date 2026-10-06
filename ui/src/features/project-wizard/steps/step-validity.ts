import { StepId, WizardState, isValidProjectName } from '../types';

import { isValidCredential } from './credential-validation';
import { isValidPolicy } from './policy-validation';
import { isValidStage } from './stage-validation';
import { isValidWarehouse } from './warehouse-validation';

// Every step but the last owns a slice of the draft. Review owns none: it can
// only be blocked by an earlier step's data, so it is not a key below.
export type DataStepId = Exclude<StepId, 'review'>;

// `reason` states the step's requirement unconditionally; callers show it only
// when `valid` is false.
export type StepValidity = { valid: boolean; reason: string };

// Whether each step's data is complete, and what it needs when it isn't. One
// source for both the per-step Continue gate and the creation gate on Review,
// which would otherwise drift apart.
export const stepDataValidity = (state: WizardState): Record<DataStepId, StepValidity> => ({
  basics: {
    valid: isValidProjectName(state.basics.name),
    reason: 'Provide a valid project name to continue'
  },
  credentials: {
    valid: state.credentials.every(isValidCredential),
    reason: 'Every credential needs a valid name, repository URL, and complete auth fields'
  },
  warehouses: {
    valid: state.warehouses.every(isValidWarehouse),
    reason: 'Every warehouse needs a valid name'
  },
  stages: {
    valid: state.stages.every(isValidStage),
    reason: 'Every stage needs a valid name and at least one requested Freight'
  },
  policies: {
    valid: state.policies.every(isValidPolicy),
    reason: 'Every policy needs a target Stage, pattern, or label'
  }
});

// The earliest step whose data is incomplete, or undefined when the draft is
// ready to apply. Creation needs this because a per-step gate only runs while
// the user passes *through* that step: the sidebar jumps straight to Review, and
// a resumed draft arrives with its credential secrets stripped. Without it, Step
// 6 would apply a Secret with a blank password, or a Warehouse with no
// subscriptions, and fail partway through the batch.
//
// An empty optional step is complete -- `[].every()` is true -- so skipping
// steps still reaches Review.
export const firstIncompleteStep = (
  state: WizardState
): { id: DataStepId; reason: string } | undefined => {
  // The Record's keys are declared in wizard order above and Object.entries
  // follows declaration order, so the earliest unmet requirement is found first.
  // The non-partial Record is what guarantees no step is missed.
  const found = Object.entries(stepDataValidity(state)).find(([, validity]) => !validity.valid);
  return found && { id: found[0] as DataStepId, reason: found[1].reason };
};
