import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

export const humanComprehendableArtifact = (
  artifact: GitCommit | Chart | Image | ArtifactReference
) => {
  if ('repoURL' in artifact) {
    const repoURL = artifact.repoURL;

    const parts = repoURL?.split('/') || [];
    const lastPart = parts[parts.length - 1];

    return lastPart;
  }

  if ('version' in artifact) {
    return artifact.version;
  }

  return '';
};

export const artifactBase = (repoURL: string) => {
  const parts = repoURL.split('/');

  return parts.slice(0, parts.length - 1).join('/');
};

export const artifactURL = (repoURL: string) =>
  repoURL.startsWith('https://') ? repoURL : `https://${repoURL}`;
