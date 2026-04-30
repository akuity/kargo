type gitOpenPrOutput = {
  pr: {
    id: string;
    url: string;
  };
};

// Steps that can produce PR deep links
export const PR_STEP_TYPES = ['git-open-pr', 'git-wait-for-pr'];

export const getPullRequestLink = (promotionStepOutput: Record<string, unknown>) =>
  (promotionStepOutput as gitOpenPrOutput)?.pr?.url;

export const getGitProviderLink = (promotionStepOutput: Record<string, unknown>) => {
  const prLink = getPullRequestLink(promotionStepOutput);

  if (!prLink) {
    return '';
  }

  try {
    const url = new URL(prLink);

    const [, owner, repo] = url.pathname.split('/');

    if (!owner || !repo) {
      throw new Error('Plugin error: missing owner or repo');
    }

    return `${url.origin}/${owner}/${repo}`;
  } catch (e) {
    // eslint-disable-next-line no-console
    console.error(e);
    return '';
  }
};
