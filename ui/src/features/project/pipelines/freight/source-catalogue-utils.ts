import { isAfter } from 'date-fns';

import { Freight } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

export const catalogueFreights = (freights: Freight[]) => {
  const catalogue = {
    images: new Set<string>(),
    commits: new Set<string>(),
    charts: new Set<string>()
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

export const catalogueFreightVersions = (freights: Freight[]) => {
  const catalogue = {
    images: new Set<string>(),
    commits: new Set<string>(),
    charts: new Set<string>()
  };

  for (const freight of freights) {
    if (freight?.images?.length) {
      for (const image of freight.images) {
        catalogue.images.add(image.tag);
      }
    }

    if (freight?.commits?.length) {
      for (const commit of freight.commits) {
        catalogue.commits.add(commit.id);
      }
    }

    if (freight?.charts?.length) {
      for (const chart of freight.charts) {
        catalogue.charts.add(chart.version);
      }
    }
  }

  return catalogue;
};

export const filterFreightByVersion = (versions: string[]) => (_freight: Freight) => {
  const freight = { ..._freight };

  if (!versions.length) {
    return freight;
  }

  freight.images = freight.images?.filter((image) => versions.includes(image.tag));
  freight.commits = freight.commits?.filter((commit) => versions.includes(commit.tag));
  freight.charts = freight.charts?.filter((chart) => versions.includes(chart.version));

  if (!freight.images.length && !freight.charts.length && !freight.commits.length) {
    return null;
  }

  return freight;
};

export const filterFreightBySource = (repoURLs: string[]) => (_freight: Freight) => {
  // clone is must
  const freight = { ..._freight };

  if (repoURLs.includes('') || !repoURLs.length) {
    return freight;
  }

  freight.images = freight.images?.filter((image) => repoURLs.includes(image.repoURL));

  freight.commits = freight.commits?.filter((commit) => repoURLs.includes(commit.repoURL));

  freight.charts = freight.charts?.filter((chart) => repoURLs.includes(chart.repoURL));

  if (!freight.images.length && !freight.charts.length && !freight.commits.length) {
    return null;
  }

  return freight;
};

export const filterFreightByTimerange = (till: Date) => (freight: Freight) => {
  const creationTimestamp = timestampDate(freight.metadata?.creationTimestamp);

  if (!creationTimestamp) {
    return false;
  }

  return isAfter(creationTimestamp, till);
};
