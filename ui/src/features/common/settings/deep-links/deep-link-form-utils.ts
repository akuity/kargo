import { z } from 'zod';

import { DeepLink } from '@ui/gen/api/v2/models';
import { zodValidators } from '@ui/utils/validators';

/**
 * Counts Go template delimiters. A URL whose `{{` and `}}` do not pair up fails
 * to parse on the server, and a link that fails to parse is simply left out of
 * the response, so the typo is worth catching before it is saved.
 */
const templateDelimitersBalanced = (value: string) =>
  (value.match(/{{/g) ?? []).length === (value.match(/}}/g) ?? []).length;

export const deepLinkFormSchema = z.object({
  title: zodValidators.requiredString,
  url: zodValidators.requiredString.refine(templateDelimitersBalanced, {
    error: 'Unbalanced {{ }}. Every {{ needs a matching }}.'
  }),
  description: z.string(),
  // conditions are expressions, not templates: the two syntaxes look close
  // enough that wrapping one in the other is an easy mistake to make
  if: z.string().refine((value) => !value.includes('{{'), {
    error: 'A condition is an expression, not a template. Drop the {{ }} wrapper.'
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
