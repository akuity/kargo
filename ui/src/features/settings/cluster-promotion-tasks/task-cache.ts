import { QueryClient } from '@tanstack/react-query';

import {
  getListClusterPromotionTasksQueryKey,
  listClusterPromotionTasksResponse
} from '@ui/gen/api/v2/core/core';
import {
  ClusterPromotionTask,
  CreateOrUpdateResourceResult,
  CreateResourceResult,
  DeleteResourceResult
} from '@ui/gen/api/v2/models';

type ResourceResult = CreateResourceResult & CreateOrUpdateResourceResult & DeleteResourceResult;

// The write response is newer than an informer-backed LIST. Do not immediately
// replace confirmed changes with a refetch that can still contain the old list.
export const updateClusterPromotionTaskCache = async (
  client: QueryClient,
  results: ResourceResult[] = []
) => {
  const changes = results.flatMap((result) => {
    if (result.error) return [];
    const manifest =
      result.updatedResourceManifest ||
      result.createdResourceManifest ||
      result.deletedResourceManifest;
    if (
      !manifest ||
      manifest.kind !== 'ClusterPromotionTask' ||
      !manifest.metadata ||
      typeof manifest.metadata !== 'object' ||
      !('name' in manifest.metadata) ||
      typeof manifest.metadata.name !== 'string'
    ) {
      return [];
    }
    return [
      {
        task: manifest as unknown as ClusterPromotionTask,
        removed: !!result.deletedResourceManifest
      }
    ];
  });
  if (changes.length === 0) return;

  const queryKey = getListClusterPromotionTasksQueryKey();
  await client.cancelQueries({ queryKey, exact: true });
  client.setQueryData<listClusterPromotionTasksResponse>(queryKey, (old) => {
    let items = [...(old?.data.items || [])];
    for (const { task, removed } of changes) {
      items = items.filter((item) => item.metadata?.name !== task.metadata?.name);
      if (!removed) items.push(task);
    }
    items.sort((a, b) => (a.metadata?.name || '').localeCompare(b.metadata?.name || ''));
    return {
      ...(old || { status: 200 as const, headers: new Headers() }),
      data: { ...old?.data, items }
    };
  });
};
