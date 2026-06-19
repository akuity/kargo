// humanComprehendableArtifact returns the last path segment of a repository URL
// (e.g. "my-app" from "ghcr.io/acme/my-app") for display.
export const humanComprehendableArtifact = (artifact: { repoURL?: string }) => {
  const parts = artifact.repoURL?.split('/') || [];

  return parts[parts.length - 1];
};

export const artifactBase = (repoURL: string) => {
  const parts = repoURL.split('/');

  return parts.slice(0, parts.length - 1).join('/');
};

export const artifactURL = (repoURL: string) =>
  repoURL.startsWith('https://') ? repoURL : `https://${repoURL}`;
