import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

export const isArtifactImage = (
  artifact: Image | Chart | GitCommit | ArtifactReference
): artifact is Image => 'digest' in artifact;

export const isArtifactChart = (
  artifact: Image | Chart | GitCommit | ArtifactReference
): artifact is Chart => 'version' in artifact;

export const isArtifactGitCommit = (
  artifact: Image | Chart | GitCommit | ArtifactReference
): artifact is GitCommit => 'id' in artifact;

export const isArtifactGeneric = (
  artifact: Image | Chart | GitCommit | ArtifactReference
): artifact is ArtifactReference => 'subscriptionName' in artifact && 'version' in artifact;
