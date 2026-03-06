import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { zodValidators } from '@ui/utils/validators';

const imageNameRegex =
  /^(?![a-zA-Z][a-zA-Z0-9+.-]*:\/\/)(\w+([.-]\w+)*(:\d+)?\/)?(\w+([.-]\w+)*)(\/\w+([.-]\w+)*)*$/;

export const createFormSchema = (genericCreds: boolean, editing?: boolean) => {
  let schema = z
    .object({
      name: zodValidators.requiredString.regex(
        dnsRegex,
        'Credentials name must be a valid DNS subdomain.'
      ),
      description: z.string().optional(),
      type: zodValidators.requiredString,
      repoUrl: zodValidators.requiredString,
      repoUrlIsRegex: z.boolean().optional(),
      username: zodValidators.requiredString,
      password: editing ? z.string().optional() : zodValidators.requiredString
    })
    .check(
      z.refine(
        (data) => {
          if (data.type === 'git' && !data.repoUrlIsRegex) {
            try {
              new URL(data.repoUrl);
            } catch {
              return false;
            }
          }

          return true;
        },
        { error: 'Repo URL must be a valid git URL.', path: ['repoUrl'] }
      ),
      z.refine(
        (data) => {
          if (data.type === 'helm' && !data.repoUrlIsRegex) {
            try {
              const url = new URL(data.repoUrl);
              if (
                url.protocol !== 'http:' &&
                url.protocol !== 'https:' &&
                url.protocol !== 'oci:'
              ) {
                return false;
              }
            } catch {
              return false;
            }
          }
          return true;
        },
        {
          error: 'Repo URL must be a valid Helm chart repository.',
          path: ['repoUrl']
        }
      ),
      z.refine(
        (data) => {
          if (data.type === 'image' && !data.repoUrlIsRegex) {
            return imageNameRegex.test(data.repoUrl);
          }
          return true;
        },
        {
          error: 'Repo URL must be a valid container registry.',
          path: ['repoUrl']
        }
      )
    );

  if (genericCreds) {
    // @ts-expect-error err
    schema = z.object({
      name: zodValidators.requiredString.regex(
        dnsRegex,
        'Credentials name must be a valid DNS subdomain.'
      ),
      description: z.string().optional(),
      type: zodValidators.requiredString,
      data: z.array(z.array(z.string()))
    });
  }

  return schema.refine((data) => ['git', 'helm', 'image', 'generic'].includes(data.type), {
    error: "Type must be one of 'git', 'helm', 'image' or 'generic'."
  });
};
