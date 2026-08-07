import { isArtifactGeneric } from '@ui/features/assemble-freight/artifact-type-guards';
import {
  ArtifactReference,
  Chart,
  FreightReference,
  GitCommit,
  Image
} from '@ui/gen/api/v2/models';

import { selectFirstArtifact as _selectFirstArtifact } from '../freight/artifact-selector-utils';

export type ArtifactTypes = Image | Chart | GitCommit | ArtifactReference;

export const normalizeFreight = (freight: FreightReference) => {
  const artifacts: ArtifactTypes[] = [];

  for (const image of freight?.images || []) {
    artifacts.push(image);
  }

  for (const commit of freight?.commits || []) {
    artifacts.push(commit);
  }

  for (const chart of freight?.charts || []) {
    artifacts.push(chart);
  }

  for (const other of freight?.artifacts || []) {
    artifacts.push(other);
  }

  return artifacts;
};

export const selectFirstArtifact = _selectFirstArtifact;

const isSameArtifact = (a: ArtifactTypes, b: ArtifactTypes) => {
  if (isArtifactGeneric(a) && isArtifactGeneric(b)) {
    return a.subscriptionName === b.subscriptionName;
  }

  if (!isArtifactGeneric(a) && !isArtifactGeneric(b)) {
    return a.repoURL === b.repoURL;
  }

  return false;
};

export const selectNextArtifact = (freight: FreightReference, current: ArtifactTypes) => {
  const artifacts = normalizeFreight(freight);
  const currentIndex = artifacts.findIndex((a) => isSameArtifact(a, current));

  if (currentIndex === -1) {
    return artifacts[0];
  }

  const nextIndex = (currentIndex + 1) % artifacts.length;
  return artifacts[nextIndex];
};

export const selectPreviousArtifact = (freight: FreightReference, current: ArtifactTypes) => {
  const artifacts = normalizeFreight(freight);
  const currentIndex = artifacts.findIndex((a) => isSameArtifact(a, current));

  if (currentIndex === -1) {
    return artifacts[artifacts.length - 1];
  }

  const previousIndex = (currentIndex - 1 + artifacts.length) % artifacts.length;
  return artifacts[previousIndex];
};
