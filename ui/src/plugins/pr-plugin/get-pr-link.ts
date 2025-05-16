export const getPullRequestLink = (promotionStepOutput: Record<string, unknown>) =>
  (promotionStepOutput as { pr: { url: string } }).pr.url;
