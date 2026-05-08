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

    const da = warehouse.status?.discoveredArtifacts;
    const images: ImageDiscoveryResult[] = [...(da?.images || [])];
    const charts: ChartDiscoveryResult[] = [...(da?.charts || [])];
    const git: GitDiscoveryResult[] = [...(da?.git || [])];

    for (const image of cloneFreightData.images || []) {
      const sub = images.find((s) => s.repoURL === image.repoURL);
      if (!sub) {
        images.push({
          repoURL: image.repoURL,
          references: [{ tag: image.tag, digest: image.digest, annotations: image.annotations }]
        } as unknown as ImageDiscoveryResult);
      } else if (!sub.references.find((r) => r.tag === image.tag)) {
        sub.references.unshift({
          tag: image.tag,
          digest: image.digest,
          annotations: image.annotations
        } as DiscoveredImageReference);
      }
    }

    for (const chart of cloneFreightData.charts || []) {
      const sub = charts.find((s) => s.repoURL === chart.repoURL && s.name === chart.name);
      if (!sub) {
        charts.push({
          $typeName: 'github.com.akuity.kargo.api.v1alpha1.ChartDiscoveryResult',
          repoURL: chart.repoURL,
          name: chart.name,
          versions: [chart.version]
        } as unknown as ChartDiscoveryResult);
      } else if (!sub.versions.find((v) => v === chart.version)) {
        sub.versions.unshift(chart.version);
      }
    }

    for (const commit of cloneFreightData.commits || []) {
      const sub = git.find((s) => s.repoURL === commit.repoURL);
      if (!sub) {
        git.push({
          repoURL: commit.repoURL,
          commits: [
            {
              id: commit.id,
              branch: commit.branch,
              tag: commit.tag,
              author: commit.author,
              committer: commit.committer,
              subject: commit.message
            }
          ]
        } as unknown as GitDiscoveryResult);
      } else if (!sub.commits.find((c) => c.id === commit.id && c.tag === commit.tag)) {
        sub.commits.unshift({
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
        discoveredArtifacts: { ...da, images, charts, git }
      }
    } as WarehouseExpanded;
  }, [warehouse, cloneFreightData]);
