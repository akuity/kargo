import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

// to preserve the order of the artifacts in the carousel, we need to use the same order everywhere
export const normalizeRepoURL = (freight: Freight) => {
  const repoURLs = [];

  if (freight?.images?.length) {
    repoURLs.push(...freight.images.map((image) => image.repoURL));
  }

  if (freight?.commits?.length) {
    repoURLs.push(...freight.commits.map((commit) => commit.repoURL));
  }

  if (freight?.charts?.length) {
    repoURLs.push(...freight.charts.map((chart) => chart.repoURL));
  }

  return repoURLs;
};

export const selectFirstArtifact = (freights: Freight[]) => {
  if (!freights?.length) {
    return '';
  }

  for (const freight of freights) {
    if (freight?.images?.length) {
      return freight?.images?.[0];
    }

    if (freight?.commits?.length) {
      return freight?.commits?.[0];
    }

    if (freight?.charts?.length) {
      return freight?.charts?.[0];
    }
  }

  return '';
};

export const selectActiveCarouselFreight = (freight: Freight, repoURL: string) => {
  return (
    freight?.images?.find((image) => image.repoURL === repoURL) ||
    freight?.commits?.find((commit) => commit.repoURL === repoURL) ||
    freight?.charts?.find((chart) => chart.repoURL === repoURL) ||
    selectFirstArtifact([freight])
  );
};

export const selectNextArtifact = (freight: Freight, repoURL: string) => {
  const repoURLs = normalizeRepoURL(freight);
  const currentIndex = repoURLs.indexOf(repoURL);

  if (currentIndex === -1) {
    return repoURLs[0];
  }

  const nextIndex = (currentIndex + 1) % repoURLs.length;
  return repoURLs[nextIndex];
};

export const selectPreviousArtifact = (freight: Freight, repoURL: string) => {
  const repoURLs = normalizeRepoURL(freight);
  const currentIndex = repoURLs.indexOf(repoURL);

  if (currentIndex === -1) {
    return repoURLs[repoURLs.length - 1];
  }

  const previousIndex = (currentIndex - 1 + repoURLs.length) % repoURLs.length;
  return repoURLs[previousIndex];
};
