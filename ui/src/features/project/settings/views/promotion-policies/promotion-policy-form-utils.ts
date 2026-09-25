import { z } from 'zod';

import {
  selectorFromValues,
  selectorLines,
  selectorSchema,
  selectorValues
} from '@ui/features/common/selector/selector-utils';
import { AutoRollbackConfig, PromotionPolicy } from '@ui/gen/api/v2/models';

/**
 * Terminal Promotion phases the API accepts as auto-rollback triggers. Note the
 * deliberate asymmetry with the verification phases below -- "Errored" here,
 * "Error" there. The CRD validates both spellings separately.
 */
export const promotionPhases = ['Failed', 'Errored'] as const;

export const verificationPhases = ['Failed', 'Error'] as const;

/**
 * What the API falls back to when auto-rollback is configured but no
 * verification phases are listed.
 */
const defaultOnVerification: VerificationPhase[] = ['Failed'];

type VerificationPhase = (typeof verificationPhases)[number];

export const promotionPolicyFormSchema = z
  .object({
    stage: selectorSchema,
    autoPromotionEnabled: z.boolean(),
    autoRollbackEnabled: z.boolean(),
    onPromotion: z.array(z.enum(promotionPhases)),
    onVerification: z.array(z.enum(verificationPhases))
  })
  .superRefine((values, ctx) => {
    // the API reads an empty list as [Failed], so leaving every box unchecked
    // would save something other than what the form shows
    if (values.autoRollbackEnabled && !values.onVerification.length) {
      ctx.addIssue({
        code: 'custom',
        message: 'Select at least one. With none selected, rollback falls back to Failed.',
        path: ['onVerification']
      });
    }
  });

export type PromotionPolicyFormValues = z.infer<typeof promotionPolicyFormSchema>;

/**
 * A policy that still targets its Stage through the deprecated `stage` field
 * rather than a `stageSelector`. The two are mutually exclusive, so such a
 * policy cannot be round-tripped through this form without rewriting it.
 */
export const isLegacyPolicy = (policy: PromotionPolicy) => !!policy.stage;

const phasesOf = <T extends string>(allowed: readonly T[], phases?: string[]): T[] =>
  (phases ?? []).filter((phase): phase is T => (allowed as readonly string[]).includes(phase));

export const formValuesFromPromotionPolicy = (
  policy?: PromotionPolicy
): PromotionPolicyFormValues => {
  const onVerification = phasesOf(verificationPhases, policy?.autoRollback?.onVerification);

  return {
    stage: selectorValues(policy?.stageSelector),
    // on for a new policy; an existing one without the field keeps the API's
    // default of off
    autoPromotionEnabled: policy ? !!policy.autoPromotionEnabled : true,
    autoRollbackEnabled: !!policy?.autoRollback,
    onPromotion: phasesOf(promotionPhases, policy?.autoRollback?.onPromotion),
    onVerification: onVerification.length ? onVerification : defaultOnVerification
  };
};

type PromotionPolicyFromFormValuesOptions = {
  /**
   * Whether the form was allowed to edit auto-rollback. When it was not, any
   * auto-rollback already on the policy is carried over untouched rather than
   * dropped -- the user never saw the field, so they never asked to clear it.
   */
  autoRollbackEditable: boolean;
  /** The policy being edited, if this is an edit rather than a create. */
  existing?: PromotionPolicy;
};

export const promotionPolicyFromFormValues = (
  values: PromotionPolicyFormValues,
  { autoRollbackEditable, existing }: PromotionPolicyFromFormValuesOptions
): PromotionPolicy => {
  let autoRollback: AutoRollbackConfig | undefined = existing?.autoRollback;

  if (autoRollbackEditable) {
    autoRollback = values.autoRollbackEnabled
      ? {
          ...(values.onPromotion.length ? { onPromotion: values.onPromotion } : {}),
          onVerification: values.onVerification
        }
      : undefined;
  }

  return {
    // an empty selector is legal and means "every Stage in the project" -- what
    // it may not be is absent, since exactly one of stage/stageSelector is required
    stageSelector: selectorFromValues(values.stage) ?? {},
    autoPromotionEnabled: values.autoPromotionEnabled,
    ...(autoRollback ? { autoRollback } : {})
  };
};

/**
 * A short, human-readable stand-in for a policy's identity. Policies have no
 * name field, so they are described by whatever they target.
 */
export const promotionPolicyLabel = (policy: PromotionPolicy) => {
  if (policy.stage) {
    return policy.stage;
  }

  const lines = selectorLines(policy.stageSelector);

  return lines.length ? lines.join(', ') : 'every Stage';
};
