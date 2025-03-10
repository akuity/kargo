import gitUrlParse from 'git-url-parse';

import { PromotionStep } from '@ui/gen/api/v1alpha1/generated_pb';
import { getPromotionStepConfig } from '@ui/plugins/atoms/plugin-helper';

export const getPullRequestLink = (
  promotionStep: PromotionStep,
  promotionStepOutput: Record<string, unknown>
) => {
  const stepConfig = getPromotionStepConfig(promotionStep);

  // plugins responsibility to keep up to date with actual promotion step config
  const provider =
    (stepConfig?.provider as 'github' | 'gitlab') || 'github'; /* default provider is github */

  const repoURL = stepConfig?.repoURL as string;

  const prNumber = promotionStepOutput?.prNumber;

  if (typeof prNumber !== 'number' || !repoURL) {
    // TODO: Throw plugin identified error to handle it at plugin level
    throw new Error(
      `cannot generate pull request link, either promotion step didn't return proper output or promotion step config is corrupted`
    );
  }

  const parsedUrl = gitUrlParse(repoURL);

  const url = `${parsedUrl.protocol}://${parsedUrl.resource}/${parsedUrl.full_name}`;

  if (provider === 'github') {
    return `${url}/pull/${prNumber}`;
  }

  if (provider === 'gitlab') {
    return `${url}/merge_requests/${prNumber}`;
  }

  throw new Error(`provider ${provider} not supported by this plugin`);
};
