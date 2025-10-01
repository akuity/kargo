import { z } from 'zod';

import { dnsRegex } from '@ui/features/common/utils';
import { zodValidators } from '@ui/utils/validators';

const helmChartRegex =
  /^(oci:\/\/)?([a-z0-9]+(?:[._-][a-z0-9]+)*\/)*[a-z0-9]+(?:[._-][a-z0-9]+)*$/i;

const imageNameRegex = /^([a-z0-9]+(?:[._-][a-z0-9]+)*\/)*[a-z0-9]+(?:[._-][a-z0-9]+)*$/i;

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
          if (data.type === 'git') {
            try {
              new URL(data.repoUrl);
            } catch {
              return false;
            }
          }

          return true;
        },
        { error: 'repoUrl must be a valid HTTPS URL.', path: ['repoUrl'] }
      ),
      z.refine(
        (data) => {
          if (data.type === 'helm') {
            return helmChartRegex.test(data.repoUrl);
          }
          return true;
        },
        {
          error: 'repoUrl must be a valid Helm chart reference.',
          path: ['repoUrl']
        }
      ),
      z.refine(
        (data) => {
          if (data.type === 'image') {
            return imageNameRegex.test(data.repoUrl);
          }
          return true;
        },
        {
          error: 'repoUrl must be a valid image name.',
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
