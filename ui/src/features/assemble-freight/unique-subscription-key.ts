// Images, git commits, and their discovery results each correspond to a single
// repository, so the repoURL identifies their subscription.
export const getArtifactSubscriptionKey = (artifact: { repoURL?: string }) =>
  artifact.repoURL || '';

// Classic (HTTP/S) chart repositories can contain differently named charts, so a
// chart's subscription key is keyed by both repoURL and name.
export const getChartSubscriptionKey = (artifact: { repoURL?: string; name?: string }) =>
  `${artifact.repoURL}/${artifact.name}`;

// Generic artifacts are keyed by the name of the subscription that discovered them.
export const getGenericSubscriptionKey = (artifact: { name?: string }) => artifact.name || '';
