import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { zodValidators } from '@ui/utils/validators';

const imageNameRegex =
  /^(?![a-zA-Z][a-zA-Z0-9+.-]*:\/\/)(\w+([.-]\w+)*(:\d+)?\/)?(\w+([.-]\w+)*)(\/\w+([.-]\w+)*)*$/;

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
    .refine(
      (data) => {
        if (!genericCreds && data.type === 'git' && data.repoUrl && !data.repoUrlIsRegex) {
          try {
            new URL(data.repoUrl);
          } catch {
            return false;
          }
        }
        return true;
      },
      { error: 'Repo URL must be a valid git URL.', path: ['repoUrl'] }
    )
    .refine(
      (data) => {
        if (!genericCreds && data.type === 'helm' && data.repoUrl && !data.repoUrlIsRegex) {
          try {
            const url = new URL(data.repoUrl);
            if (url.protocol !== 'http:' && url.protocol !== 'https:' && url.protocol !== 'oci:') {
              return false;
            }
          } catch {
            return false;
          }
        }
        return true;
      },
      { error: 'Repo URL must be a valid Helm chart repository.', path: ['repoUrl'] }
    )
    .refine(
      (data) => {
        if (!genericCreds && data.type === 'image' && data.repoUrl && !data.repoUrlIsRegex) {
          return imageNameRegex.test(data.repoUrl);
        }
        return true;
      },
      { error: 'Repo URL must be a valid container registry.', path: ['repoUrl'] }
    )
    .refine((data) => ['git', 'helm', 'image', 'generic'].includes(data.type), {
      error: "Type must be one of 'git', 'helm', 'image' or 'generic'."
    });
