export const humanComprehendableArtifact = (repoURL: string) => {
  const parts = repoURL.split('/');
  const lastPart = parts[parts.length - 1];

  return lastPart;
};

export const artifactBase = (repoURL: string) => {
  const parts = repoURL.split('/');

  return parts.slice(0, parts.length - 1).join('/');
};

export const artifactURL = (repoURL: string) =>
  repoURL.startsWith('https://') ? repoURL : `https://${repoURL}`;
