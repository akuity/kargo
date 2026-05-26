import { Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

export const isArtifactImage = (artifact: Image | Chart | GitCommit): artifact is Image =>
  'digest' in artifact;

export const isArtifactChart = (artifact: Image | Chart | GitCommit): artifact is Chart =>
  'version' in artifact;

export const isArtifactGitCommit = (artifact: Image | Chart | GitCommit): artifact is GitCommit =>
  'id' in artifact;
