import { RepoSubscription, WarehouseExpanded } from '@ui/extend/types';
import { ArtifactReference, Chart, GitCommit, Image } from '@ui/gen/api/v2/models';

import { humanComprehendableArtifact } from './artifact-parts-utils';

// findSubscription finds the Warehouse subscription to an artifact's
// repository. A classic Helm repository serves many charts from one URL, so a
// chart must match the subscription's chart name too, where it has one.
const findSubscription = (
  repoURL: string,
  chartName: string | undefined,
  subscriptions: RepoSubscription[]
) =>
  subscriptions.find(
    (subscription) =>
      subscription.git?.repoURL === repoURL ||
      subscription.image?.repoURL === repoURL ||
      (subscription.chart?.repoURL === repoURL &&
        (!subscription.chart.name || subscription.chart.name === chartName))
  );

// getSubscriptionLabel names the Warehouse subscription an artifact came from
// as the pipeline graph's subscription node does: the name recorded on the
// artifact, else the name of its subscription in the Warehouse spec (freight
// discovered before the subscription was named does not record it), else the
// repository's short name.
export const getSubscriptionLabel = (
  artifact: GitCommit | Chart | Image | ArtifactReference,
  subscriptions: RepoSubscription[] = []
): string => {
  if (artifact.subscriptionName) {
    return artifact.subscriptionName;
  }

  if (!('repoURL' in artifact) || !artifact.repoURL) {
    return ('artifactType' in artifact && artifact.artifactType) || '';
  }

  const { repoURL } = artifact;
  const chartName = 'name' in artifact ? artifact.name : undefined;

  return (
    findSubscription(repoURL, chartName, subscriptions)?.name ||
    humanComprehendableArtifact({ repoURL }) ||
    repoURL
  );
};

// Past this many subscriptions in one Warehouse, its artifacts are hard to tell
// apart by their icons alone.
const MANY_SUBSCRIPTIONS = 4;

// getShowSubscriptionNames is whether the freight timeline names the
// subscription each artifact came from: as the user chose, or, until they
// choose, when any Warehouse has more than MANY_SUBSCRIPTIONS subscriptions.
export const getShowSubscriptionNames = (
  choice: boolean | undefined,
  warehouses: WarehouseExpanded[]
): boolean =>
  choice ??
  warehouses.some(
    (warehouse) => (warehouse?.spec?.subscriptions?.length || 0) > MANY_SUBSCRIPTIONS
  );
