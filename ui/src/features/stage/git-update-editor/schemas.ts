import { z } from 'zod';

import { zodValidators } from '@ui/utils/validators';

export const requestedFreightSchema = z.object({
  warehouse: zodValidators.requiredString,
  sources: z.object({
    direct: z.boolean(),
    upstreamStages: z.array(z.string())
  })
});

export const pullRequestMechanismSchema = z.object({
  type: z.enum(['github', 'gitlab'])
});

export const renderImageUpdateSchema = z.object({
  image: zodValidators.requiredString,
  warehouse: z.string().optional(),
  useDigest: z.boolean()
});

export const renderMechanismSchema = z.object({
  images: z.array(renderImageUpdateSchema),
  warehouse: z.string().optional()
});

export const kustomizeImageUpdateSchema = z.object({
  image: z.string(),
  warehouse: z.string(),
  path: zodValidators.requiredString,
  useDigest: z.boolean()
});

export const kustomizeMechanismSchema = z.object({
  images: z.array(kustomizeImageUpdateSchema),
  warehouse: z.string()
});

export const helmImageUpdateSchema = z.object({
  image: zodValidators.requiredString,
  warehouse: z.string(),
  valuesFilePath: zodValidators.requiredString,
  key: zodValidators.requiredString,
  value: zodValidators.requiredString
});

export const helmChartDependencyUpdateSchema = z.object({
  repository: zodValidators.requiredString,
  name: zodValidators.requiredString,
  warehouse: z.string(),
  chartPath: z.string()
});

export const helmMechanismSchema = z.object({
  images: z.array(helmImageUpdateSchema),
  charts: z.array(helmChartDependencyUpdateSchema),
  warehouse: z.string()
});

export const gitRepoUpdateSchema = z.object({
  repoUrl: zodValidators.requiredString,
  insecureSkipVerify: z.boolean(),
  readBranch: z.string(),
  writeBranch: z.string(),
  pullRequest: pullRequestMechanismSchema.optional(),
  render: renderMechanismSchema.optional(),
  kustomize: kustomizeMechanismSchema.optional(),
  helm: helmMechanismSchema.optional()
});
