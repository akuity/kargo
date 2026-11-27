import {
  isArtifactChart,
  isArtifactGitCommit,
  isArtifactImage
} from '@ui/features/assemble-freight/artifact-type-guards';
import {
  getGitCommitURL,
  getImageSource
} from '@ui/features/freight-timeline/open-container-initiative-utils';
import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { shortVersion } from './short-version-utils';

export type ArtifactDisplay = {
  // the version the tag shows
  value: string;
  // what hovering the tag shows
  title?: string;
  // where the tag links to; empty when nowhere
  url: string;
};

// getArtifactDisplay is what an artifact tag shows for an artifact: its
// version, shortened, the repository it came from on hover, and a link to its
// source where there is one.
export const getArtifactDisplay = (
  artifact: GitCommit | Chart | Image | ArtifactReference
): ArtifactDisplay => {
  if (isArtifactGitCommit(artifact)) {
    return {
      // prioritize semver; use shortVersion for tags, 7-char slice for raw commit hashes
      value: (artifact.tag ? shortVersion(artifact.tag) : artifact.id?.slice(0, 7)) || '',
      title: artifact.repoURL,
      url: getGitCommitURL(artifact.repoURL || '', artifact.id || '')
    };
  }

  if (isArtifactChart(artifact)) {
    return {
      value: shortVersion(artifact.version),
      title: `${artifact.repoURL}:${artifact.version}`,
      url: ''
    };
  }

  if (isArtifactImage(artifact)) {
    return {
      value: shortVersion(artifact.tag),
      title: `${artifact.repoURL}:${artifact.tag}`,
      url: (artifact.annotations && getImageSource(artifact.annotations)) || ''
    };
  }

  return { value: shortVersion(artifact.version), url: '' };
};
