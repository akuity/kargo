import { describe, expect, test } from 'vitest';

import { PromotionPolicy } from '@ui/gen/api/v2/models';

import {
  formValuesFromPromotionPolicy,
  isLegacyPolicy,
  promotionPolicyFormSchema,
  promotionPolicyFromFormValues,
  promotionPolicyLabel,
  PromotionPolicyFormValues
} from './promotion-policy-form-utils';

const blankSelector = {
  nameMode: 'exact' as const,
  name: '',
  labels: [],
  matchExpressions: []
};

const values = (overrides: Partial<PromotionPolicyFormValues> = {}): PromotionPolicyFormValues => ({
  stage: blankSelector,
  autoPromotionEnabled: false,
  autoRollbackEnabled: false,
  onPromotion: [],
  onVerification: ['Failed'],
  ...overrides
});

describe('promotionPolicyFormSchema', () => {
  test('requires a verification phase once auto-rollback is on', () => {
    const result = promotionPolicyFormSchema.safeParse(
      values({ autoRollbackEnabled: true, onVerification: [] })
    );

    // an empty list would save as [Failed] -- not what the unchecked boxes show
    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.path)).toEqual([['onVerification']]);
  });

  test('ignores verification phases while auto-rollback is off', () => {
    const result = promotionPolicyFormSchema.safeParse(
      values({ autoRollbackEnabled: false, onVerification: [] })
    );

    expect(result.success).toBe(true);
  });
});

describe('formValuesFromPromotionPolicy', () => {
  test('gives a blank, auto-promotion-on form for a new policy', () => {
    expect(formValuesFromPromotionPolicy()).toEqual(values({ autoPromotionEnabled: true }));
  });

  test('keeps auto-promotion off for an existing policy that does not set it', () => {
    expect(formValuesFromPromotionPolicy({ stageSelector: {} }).autoPromotionEnabled).toBe(false);
  });

  test('reads the stage selector back into selector fields', () => {
    const form = formValuesFromPromotionPolicy({
      stageSelector: { name: 'glob:prod-*', matchLabels: { tier: 'critical' } },
      autoPromotionEnabled: true
    });

    expect(form.stage.nameMode).toBe('glob');
    expect(form.stage.name).toBe('prod-*');
    expect(form.stage.labels).toEqual([{ key: 'tier', value: 'critical' }]);
    expect(form.autoPromotionEnabled).toBe(true);
  });

  test('treats a present but empty auto-rollback as enabled', () => {
    const form = formValuesFromPromotionPolicy({ stageSelector: {}, autoRollback: {} });

    expect(form.autoRollbackEnabled).toBe(true);
    // the API defaults an absent verification list to [Failed]
    expect(form.onVerification).toEqual(['Failed']);
    expect(form.onPromotion).toEqual([]);
  });

  test('keeps only phases the API accepts', () => {
    const form = formValuesFromPromotionPolicy({
      stageSelector: {},
      autoRollback: {
        onPromotion: ['Errored', 'Succeeded'],
        // "Error", not "Errored", on the verification side
        onVerification: ['Error', 'Errored']
      }
    });

    expect(form.onPromotion).toEqual(['Errored']);
    expect(form.onVerification).toEqual(['Error']);
  });
});

describe('promotionPolicyFromFormValues', () => {
  test('emits an empty selector rather than none when nothing is constrained', () => {
    const policy = promotionPolicyFromFormValues(values(), { autoRollbackEditable: true });

    // exactly one of stage/stageSelector is required, so the key has to survive
    expect(policy).toEqual({ stageSelector: {}, autoPromotionEnabled: false });
  });

  test('prefixes a non-exact name and collects label constraints', () => {
    const policy = promotionPolicyFromFormValues(
      values({
        stage: {
          nameMode: 'regex',
          name: 'prod-.*',
          labels: [{ key: 'tier', value: 'critical' }],
          matchExpressions: []
        },
        autoPromotionEnabled: true
      }),
      { autoRollbackEditable: true }
    );

    expect(policy.stageSelector).toEqual({
      name: 'regex:prod-.*',
      matchLabels: { tier: 'critical' }
    });
    expect(policy.autoPromotionEnabled).toBe(true);
  });

  test('writes auto-rollback only when it is switched on', () => {
    expect(
      promotionPolicyFromFormValues(values({ autoRollbackEnabled: false }), {
        autoRollbackEditable: true
      }).autoRollback
    ).toBeUndefined();

    expect(
      promotionPolicyFromFormValues(
        values({
          autoRollbackEnabled: true,
          onPromotion: ['Failed'],
          onVerification: ['Failed', 'Error']
        }),
        { autoRollbackEditable: true }
      ).autoRollback
    ).toEqual({ onPromotion: ['Failed'], onVerification: ['Failed', 'Error'] });
  });

  test('omits an empty promotion phase list', () => {
    const policy = promotionPolicyFromFormValues(
      values({ autoRollbackEnabled: true, onPromotion: [] }),
      { autoRollbackEditable: true }
    );

    expect(policy.autoRollback).toEqual({ onVerification: ['Failed'] });
  });

  test('carries auto-rollback through untouched when the form could not edit it', () => {
    const existing: PromotionPolicy = {
      stageSelector: { name: 'prod' },
      autoRollback: { onVerification: ['Error'] }
    };

    const policy = promotionPolicyFromFormValues(values({ autoRollbackEnabled: false }), {
      autoRollbackEditable: false,
      existing
    });

    // the user never saw the field, so they never asked to clear it
    expect(policy.autoRollback).toEqual({ onVerification: ['Error'] });
  });

  test('leaves auto-rollback off for a new policy the form could not edit it on', () => {
    expect(
      promotionPolicyFromFormValues(values(), { autoRollbackEditable: false }).autoRollback
    ).toBeUndefined();
  });
});

describe('isLegacyPolicy', () => {
  test('is true only for the deprecated stage field', () => {
    expect(isLegacyPolicy({ stage: 'prod' })).toBe(true);
    expect(isLegacyPolicy({ stageSelector: { name: 'prod' } })).toBe(false);
    expect(isLegacyPolicy({ stageSelector: {} })).toBe(false);
  });
});

describe('promotionPolicyLabel', () => {
  test('describes a policy by whatever it targets', () => {
    expect(promotionPolicyLabel({ stage: 'prod' })).toBe('prod');
    expect(promotionPolicyLabel({ stageSelector: { name: 'glob:prod-*' } })).toBe('glob:prod-*');
    expect(promotionPolicyLabel({ stageSelector: { matchLabels: { tier: 'critical' } } })).toBe(
      'tier=critical'
    );
    expect(promotionPolicyLabel({ stageSelector: {} })).toBe('every Stage');
  });
});
