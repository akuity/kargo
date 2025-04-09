import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';

export const catalogueFreights = (freights: Freight[]) => {
  const catalogue = {
    images: new Set(),
    commits: new Set(),
    charts: new Set()
  };

  for (const freight of freights) {
    if (freight?.images?.length) {
      for (const image of freight.images) {
        catalogue.images.add(image.repoURL);
      }
    }

    if (freight?.commits?.length) {
      for (const commit of freight.commits) {
        catalogue.commits.add(commit.repoURL);
      }
    }

    if (freight?.charts?.length) {
      for (const chart of freight.charts) {
        catalogue.charts.add(chart.repoURL);
      }
    }
  }

  return catalogue;
};
