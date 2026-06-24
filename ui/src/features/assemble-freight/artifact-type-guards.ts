import {
  ArtifactReference,
  Chart,
  DiscoveredCommit,
  DiscoveredImageReference,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';

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

export const isDiscoveredCommit = (
  item: DiscoveredCommit | DiscoveredImageReference
): item is DiscoveredCommit => 'id' in item;

export const isDiscoveredImage = (
  item: DiscoveredCommit | DiscoveredImageReference
): item is DiscoveredImageReference => 'tag' in item;
