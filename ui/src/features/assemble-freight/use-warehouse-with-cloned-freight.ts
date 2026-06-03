import { useMemo } from 'react';

import { WarehouseExpanded } from '@ui/extend/types';
import {
  ChartDiscoveryResult,
  DiscoveredCommit,
  DiscoveredImageReference,
  Freight,
  GitDiscoveryResult,
  ImageDiscoveryResult
} from '@ui/gen/api/v1alpha1/generated_pb';

import { getSubscriptionKey, getSubscriptionKeyFreight } from './unique-subscription-key';

// WHY: A Warehouse's discoveredArtifacts is a rolling window of recently seen
// artifact versions — it does not retain a full history. When a user wants to
// clone an existing piece of Freight (i.e. assemble a new Freight starting from
// the same artifact versions), the specific image tags, chart versions, or git
// commits referenced by that Freight may have aged out of the discovery window
// and therefore no longer appear in discoveredArtifacts.
//
// Without this injection, the assembly UI (ImageTable, CommitTable, ChartTable)
// would have no knowledge of those versions. They would not appear as selectable
// options, and mergeWithClonedFreight would fail to pre-select them because it
// looks up artifacts by repoURL in the discovered data.
//
// This hook solves that by merging the cloned Freight's artifacts back into the
// warehouse's discoveredArtifacts before the assembly UI renders:
//   - If a subscription (repoURL) is entirely absent, a synthetic DiscoveryResult
//     is created for it so the sidebar menu shows the entry at all.
//   - If the subscription exists but the specific version is missing, it is
//     prepended (unshift) to the front of the version list so it appears first
//     and is the default selection.
//
// The enriched warehouse is passed down to AssembleFreight as if the warehouse
// had always discovered those versions, giving the user a coherent "clone this
// Freight" experience even when the source versions are no longer current.
export const useWarehouseWithClonedFreight = (
  warehouse: WarehouseExpanded,
  cloneFreightData: Freight | undefined
): WarehouseExpanded =>
  useMemo(() => {
    if (!cloneFreightData) {
      return warehouse;
    }

    const discoveredArtifacts = structuredClone(warehouse.status?.discoveredArtifacts);
    const images: ImageDiscoveryResult[] = discoveredArtifacts?.images || [];
    const charts: ChartDiscoveryResult[] = discoveredArtifacts?.charts || [];
    const git: GitDiscoveryResult[] = discoveredArtifacts?.git || [];

    for (const image of cloneFreightData.images || []) {
      let subscription = images.find(
        (s) => getSubscriptionKey(s) === getSubscriptionKeyFreight(image)
      );
      if (!subscription) {
        subscription = {
          repoURL: image.repoURL,
          references: []
        } as unknown as ImageDiscoveryResult;
        images.push(subscription);
      }

      if (!subscription?.references.find((r) => r.tag === image.tag)) {
        subscription?.references.unshift({
          tag: image.tag,
          digest: image.digest,
          annotations: image.annotations
        } as DiscoveredImageReference);
      }
    }

    for (const chart of cloneFreightData.charts || []) {
      let subscription = charts.find(
        (s) => getSubscriptionKey(s) === getSubscriptionKeyFreight(chart)
      );
      if (!subscription) {
        subscription = {
          $typeName: 'github.com.akuity.kargo.api.v1alpha1.ChartDiscoveryResult',
          repoURL: chart.repoURL,
          name: chart.name,
          versions: []
        } as unknown as ChartDiscoveryResult;
        charts.push(subscription);
      }

      if (!subscription?.versions.find((v) => v === chart.version)) {
        subscription?.versions.unshift(chart.version);
      }
    }

    for (const commit of cloneFreightData.commits || []) {
      let subscription = git.find(
        (s) => getSubscriptionKey(s) === getSubscriptionKeyFreight(commit)
      );
      if (!subscription) {
        subscription = {
          repoURL: commit.repoURL,
          commits: []
        } as unknown as GitDiscoveryResult;
        git.push(subscription);
      }

      if (!subscription?.commits.find((c) => c.id === commit.id && c.tag === commit.tag)) {
        subscription?.commits.unshift({
          id: commit.id,
          branch: commit.branch,
          tag: commit.tag,
          author: commit.author,
          committer: commit.committer,
          subject: commit.message
        } as DiscoveredCommit);
      }
    }

    return {
      ...warehouse,
      status: {
        ...warehouse.status,
        discoveredArtifacts
      }
    } as WarehouseExpanded;
  }, [warehouse, cloneFreightData]);
