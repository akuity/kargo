import {
  ArtifactReference,
  Chart,
  Freight,
  FreightReference,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';
// DEFAULT_MAX_ARTIFACTS is the number of artifact tags rendered before the
// remainder is collapsed into a "+N more" indicator.
export const DEFAULT_MAX_ARTIFACTS = 3;

export type FreightArtifactItem = GitCommit | Chart | Image | ArtifactReference;

// getFreightArtifacts flattens a Freight's (or FreightReference's) commits,
// charts, images, and other artifacts into a single ordered list.
export const getFreightArtifacts = (
  freight?: Freight | FreightReference
): FreightArtifactItem[] => [
  ...(freight?.commits || []),
  ...(freight?.charts || []),
  ...(freight?.images || []),
  ...(freight?.artifacts || [])
];
