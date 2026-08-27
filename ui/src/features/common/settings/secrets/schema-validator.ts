import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { zodValidators } from '@ui/utils/validators';

export const imageNameRegex =
  /^(?![a-zA-Z][a-zA-Z0-9+.-]*:\/\/)(\w+([.-]\w+)*(:\d+)?\/)?(\w+([.-]\w+)*)(\/\w+([.-]\w+)*)*$/;

const parseUrl = (value: string): URL | undefined => {
  try {
    return new URL(value);
  } catch {
    return undefined;
  }
};

const helmProtocols = ['http:', 'https:', 'oci:'];

// The repo URL shape a credential type demands, or undefined when the URL is
// acceptable. Shared with the project wizard's credential step so both surfaces
// reject the same URLs with the same wording. A regex pattern matches URLs
// rather than being one, and types without a shape rule (generic) are left
// alone; an empty URL is a separate, "required" concern.
export const repoUrlError = (
  type: string,
  repoUrl?: string,
  repoUrlIsRegex?: boolean
): string | undefined => {
  if (!repoUrl || repoUrlIsRegex) {
    return undefined;
  }
  switch (type) {
    case 'git':
      return parseUrl(repoUrl) ? undefined : 'Repo URL must be a valid git URL.';
    case 'helm': {
      const url = parseUrl(repoUrl);
      return url && helmProtocols.includes(url.protocol)
        ? undefined
        : 'Repo URL must be a valid Helm chart repository.';
    }
    case 'image':
      return imageNameRegex.test(repoUrl)
        ? undefined
        : 'Repo URL must be a valid container registry.';
    default:
      return undefined;
  }
};

// secretFormSchema is the unified shape backing both the repo credentials form
// and the generic secret form. Repo- and generic-specific requirements are
// enforced via conditional refinements in createFormSchema, so the inferred
// type stays stable across both modes.
const secretFormSchema = z.object({
  name: zodValidators.requiredString.regex(
    dnsRegex,
    'Credentials name must be a valid DNS subdomain.'
  ),
  description: z.string().optional(),
  type: zodValidators.requiredString,
  repoUrl: z.string().optional(),
  repoUrlIsRegex: z.boolean().optional(),
  username: z.string().optional(),
  password: z.string().optional(),
  secretType: z.string().optional(),
  data: z.array(z.tuple([z.string(), z.string()])).optional(),
  replicate: z.boolean().optional()
});

export type SecretFormValues = z.infer<typeof secretFormSchema>;

export const createFormSchema = (genericCreds: boolean, editing?: boolean) =>
  secretFormSchema
    .refine((data) => genericCreds || !!data.repoUrl, {
      error: 'Repo URL is required.',
      path: ['repoUrl']
    })
    .refine((data) => genericCreds || !!data.username, {
      error: 'Username is required.',
      path: ['username']
    })
    .refine((data) => genericCreds || editing || !!data.password, {
      error: 'Password is required.',
      path: ['password']
    })
    // superRefine rather than a refine per type: the message depends on the
    // type, and only one of them can ever apply.
    .superRefine((data, ctx) => {
      if (genericCreds) {
        return;
      }
      const error = repoUrlError(data.type, data.repoUrl, data.repoUrlIsRegex);
      if (error) {
        ctx.addIssue({ code: 'custom', message: error, path: ['repoUrl'] });
      }
    })
    .refine((data) => ['git', 'helm', 'image', 'generic'].includes(data.type), {
      error: "Type must be one of 'git', 'helm', 'image' or 'generic'."
    });
