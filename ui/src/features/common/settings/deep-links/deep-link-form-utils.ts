import { z } from 'zod';

import { DeepLink } from '@ui/gen/api/v2/models';
import { zodValidators } from '@ui/utils/validators';

/**
 * Matches a `{{` that does not open a `${{` expression, i.e. Go template
 * syntax left over from before deep links used expressions. The server
 * rejects it, so it is worth catching before it is saved.
 */
const legacyTemplateSyntax = /(^|[^$]){{/;

/**
 * Counts expression delimiters. A URL whose `${{` and `}}` do not pair up fails
 * on the server, and a link that fails is simply left out of the response, so
 * the typo is worth catching before it is saved. This is a heuristic: an
 * expression that itself contains `}}`, such as a nested map literal, trips it.
 */
const expressionDelimitersBalanced = (value: string) =>
  (value.match(/\$\{\{/g) ?? []).length === (value.match(/}}/g) ?? []).length;

/**
 * The server accepts any expression that evaluates to a boolean, but a single
 * `${{ }}` expression is the one shape worth guiding people toward.
 */
const isWrappedExpression = (value: string) => {
  const trimmed = value.trim();
  return trimmed === '' || (trimmed.startsWith('${{') && trimmed.endsWith('}}'));
};

export const deepLinkFormSchema = z.object({
  title: zodValidators.requiredString,
  url: zodValidators.requiredString
    .refine((value) => !legacyTemplateSyntax.test(value), {
      error:
        'Go template syntax (e.g. {{ .stage.metadata.name }}) is no longer supported. Use expr-lang expressions offset by ${{ }} instead (e.g. ${{ stage.metadata.name }}).',
      // the delimiter count would also fail on legacy syntax; one message is enough
      abort: true
    })
    .refine(expressionDelimitersBalanced, {
      error: 'Unbalanced ${{ }}. Every ${{ needs a matching }}.'
    }),
  description: z.string(),
  if: z.string().refine(isWrappedExpression, {
    error: 'A condition must be wrapped in ${{ }}.'
  })
});

export type DeepLinkFormValues = z.infer<typeof deepLinkFormSchema>;

export const formValuesFromDeepLink = (link?: DeepLink): DeepLinkFormValues => ({
  title: link?.title ?? '',
  url: link?.url ?? '',
  description: link?.description ?? '',
  if: link?.if ?? ''
});

export const deepLinkFromFormValues = (values: DeepLinkFormValues): DeepLink => {
  const description = values.description.trim();
  const condition = values.if.trim();

  return {
    title: values.title.trim(),
    url: values.url.trim(),
    // both are optional on the CRD, so send them only when set and an untouched
    // field does not land in the manifest as an empty string
    ...(description ? { description } : {}),
    ...(condition ? { if: condition } : {})
  };
};
