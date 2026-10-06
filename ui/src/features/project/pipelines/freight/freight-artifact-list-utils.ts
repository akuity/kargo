import { faDocker, faGit } from '@fortawesome/free-brands-svg-icons';
import { faAnchor, faBox, IconDefinition } from '@fortawesome/free-solid-svg-icons';

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

// FreightArtifactCount is a per-kind tally of artifacts carried by a piece of
// Freight.
export type FreightArtifactCount = {
  icon: IconDefinition;
  count: number;
};

// getHiddenFreightArtifactCounts tallies, by kind, the artifacts a
// FreightArtifactList leaves unrendered once it has shown its first `visible`
// ones. Kinds are walked in the order getFreightArtifacts concatenates them, so
// the tally matches exactly what the list sliced off. Kinds that are fully
// visible are omitted.
export const getHiddenFreightArtifactCounts = (
  freight?: Freight | FreightReference,
  visible = DEFAULT_MAX_ARTIFACTS
): FreightArtifactCount[] => {
  const hidden: FreightArtifactCount[] = [];
  let remainingVisible = Math.max(visible, 0);

  for (const kind of [
    { icon: faGit, count: freight?.commits?.length || 0 },
    { icon: faAnchor, count: freight?.charts?.length || 0 },
    { icon: faDocker, count: freight?.images?.length || 0 },
    { icon: faBox, count: freight?.artifacts?.length || 0 }
  ]) {
    const shown = Math.min(remainingVisible, kind.count);
    remainingVisible -= shown;

    if (kind.count > shown) {
      hidden.push({ icon: kind.icon, count: kind.count - shown });
    }
  }

  return hidden;
};
