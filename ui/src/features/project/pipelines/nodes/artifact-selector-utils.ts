import { Chart, FreightReference, GitCommit, Image } from '@ui/gen/api/v1alpha1/generated_pb';

import { selectFirstArtifact as _selectFirstArtifact } from '../freight/artifact-selector-utils';

export type ArtifactTypes = Image | Chart | GitCommit;

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

  return artifacts;
};

export const selectFirstArtifact = _selectFirstArtifact;

export const selectNextArtifact = (freight: FreightReference, current: ArtifactTypes) => {
  const artifacts = normalizeFreight(freight);
  const currentIndex = artifacts.findIndex((a) => a.repoURL === current.repoURL);

  if (currentIndex === -1) {
    return artifacts[0];
  }

  const nextIndex = (currentIndex + 1) % artifacts.length;
  return artifacts[nextIndex];
};

export const selectPreviousArtifact = (freight: FreightReference, current: ArtifactTypes) => {
  const artifacts = normalizeFreight(freight);
  const currentIndex = artifacts.findIndex((a) => a.repoURL === current.repoURL);

  if (currentIndex === -1) {
    return artifacts[artifacts.length - 1];
  }

  const previousIndex = (currentIndex - 1 + artifacts.length) % artifacts.length;
  return artifacts[previousIndex];
};
