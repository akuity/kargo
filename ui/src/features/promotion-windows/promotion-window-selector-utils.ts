import { z } from 'zod';

import { PromotionPolicySelector, V1LabelSelectorRequirement } from '@ui/gen/api/v2/models';

export const valuelessOperators = ['Exists', 'DoesNotExist'];

export const expressionOperators = ['In', 'NotIn', ...valuelessOperators];

const expressionSchema = z
  .object({
    key: z.string(),
    operator: z.string(),
    values: z.array(z.string())
  })
  .superRefine((expression, ctx) => {
    if (!expression.key.trim()) {
      return;
    }
    if (!expression.operator) {
      ctx.addIssue({
        code: 'custom',
        message: 'An operator is required.',
        path: ['operator']
      });
      return;
    }
    if (!valuelessOperators.includes(expression.operator) && !expression.values.length) {
      ctx.addIssue({
        code: 'custom',
        message: `${expression.operator} requires at least one value.`,
        path: ['values']
      });
    }
  });

export const selectorSchema = z.object({
  nameMode: z.enum(['exact', 'glob', 'regex']),
  name: z.string(),
  labels: z.array(z.object({ key: z.string(), value: z.string() })),
  matchExpressions: z.array(expressionSchema)
});

export type SelectorValues = z.infer<typeof selectorSchema>;

export const selectorValues = (selector?: PromotionPolicySelector): SelectorValues => {
  const [, prefix, pattern] = /^(glob|regex|regexp):(.*)$/.exec(selector?.name ?? '') ?? [];

  return {
    nameMode: !prefix ? 'exact' : prefix === 'glob' ? 'glob' : 'regex',
    name: pattern ?? selector?.name ?? '',
    labels: Object.entries(selector?.matchLabels ?? {}).map(([key, value]) => ({ key, value })),
    matchExpressions: (selector?.matchExpressions ?? []).map((expression) => ({
      key: expression.key ?? '',
      operator: expression.operator ?? '',
      values: expression.values ?? []
    }))
  };
};

export const selectorFromValues = (values: SelectorValues): PromotionPolicySelector | undefined => {
  const name = values.name.trim();
  const matchLabels = Object.fromEntries(
    values.labels
      .filter((label) => label.key.trim())
      .map((label) => [label.key.trim(), label.value.trim()])
  );
  const matchExpressions = values.matchExpressions
    .filter((expression) => expression.key.trim() && expression.operator)
    .map((expression) => ({
      key: expression.key.trim(),
      operator: expression.operator,
      ...(valuelessOperators.includes(expression.operator) ? {} : { values: expression.values })
    }));

  const selector: PromotionPolicySelector = {
    ...(name ? { name: values.nameMode === 'exact' ? name : `${values.nameMode}:${name}` } : {}),
    ...(Object.keys(matchLabels).length ? { matchLabels } : {}),
    ...(matchExpressions.length ? { matchExpressions } : {})
  };

  return Object.keys(selector).length ? selector : undefined;
};

const expressionText = (requirement: V1LabelSelectorRequirement) => {
  const key = requirement.key ?? '';
  const values = (requirement.values ?? []).join(', ');

  switch (requirement.operator) {
    case 'In':
      return `${key} in (${values})`;
    case 'NotIn':
      return `${key} notin (${values})`;
    case 'Exists':
      return `${key} exists`;
    case 'DoesNotExist':
      return `${key} does not exist`;
    default:
      return [key, requirement.operator, values].filter(Boolean).join(' ');
  }
};

export const selectorLines = (selector?: PromotionPolicySelector): string[] =>
  [
    selector?.name,
    ...Object.entries(selector?.matchLabels ?? {}).map(([key, value]) => `${key}=${value}`),
    ...(selector?.matchExpressions ?? []).map(expressionText)
  ].filter((line): line is string => !!line);
