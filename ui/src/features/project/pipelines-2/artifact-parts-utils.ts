export const humanComprehendableArtifact = (repoURL: string) => {
  const parts = repoURL.split('/');
  const lastPart = parts[parts.length - 1];

  return lastPart;
};
