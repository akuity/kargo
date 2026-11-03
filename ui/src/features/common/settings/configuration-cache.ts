import { QueryClient } from '@tanstack/react-query';

import {
  getGetProjectConfigQueryKey,
  getGetProjectConfigQueryOptions
} from '@ui/gen/api/v2/core/core';
import { CreateOrUpdateResourceResult } from '@ui/gen/api/v2/models';
import {
  getGetClusterConfigQueryKey,
  getGetClusterConfigQueryOptions
} from '@ui/gen/api/v2/system/system';
import { ApiError } from '@ui/lib/api/custom-fetch';

// Every settings editor writes the same complete resource. Share a mutation
// scope so each save starts after the previous confirmed response is cached.
export const configurationMutationScope = (project: string | undefined) => ({
  id: JSON.stringify(
    project === undefined ? getGetClusterConfigQueryKey() : getGetProjectConfigQueryKey(project)
  )
});

export const configurationForUpdate = async (client: QueryClient, project: string | undefined) => {
  try {
    // ensureQueryData reuses a confirmed write instead of replacing it with an
    // informer read that can still be one save behind. Fetch only on a cold cache.
    if (project === undefined) {
      return (
        await client.ensureQueryData(
          getGetClusterConfigQueryOptions({ query: { meta: { silent404: true } } })
        )
      ).data;
    }
    return (
      await client.ensureQueryData(
        getGetProjectConfigQueryOptions(project, { query: { meta: { silent404: true } } })
      )
    ).data;
  } catch (error) {
    if (error instanceof ApiError && error.isNotFound()) return undefined;
    throw error;
  }
};

// Keep the confirmed complete configuration, including unrelated settings.
// An immediate informer-backed refetch can still return the pre-save version.
export const updateConfigurationCache = async (
  client: QueryClient,
  project: string | undefined,
  results: CreateOrUpdateResourceResult[] = []
) => {
  const kind = project === undefined ? 'ClusterConfig' : 'ProjectConfig';
  const saved = results
    .filter((result) => !result.error)
    .map((result) => result.updatedResourceManifest || result.createdResourceManifest)
    .find((manifest) => {
      if (
        !manifest ||
        manifest.kind !== kind ||
        !manifest.metadata ||
        typeof manifest.metadata !== 'object'
      )
        return false;
      const metadata = manifest.metadata as { name?: string; namespace?: string };
      return (
        metadata.name === (project ?? 'cluster') &&
        (project === undefined ? !metadata.namespace : metadata.namespace === project)
      );
    });
  if (!saved) return;
  const queryKey =
    project === undefined ? getGetClusterConfigQueryKey() : getGetProjectConfigQueryKey(project);
  await client.cancelQueries({ queryKey, exact: true });
  client.setQueryData(queryKey, { data: saved, status: 200, headers: new Headers() });
};
