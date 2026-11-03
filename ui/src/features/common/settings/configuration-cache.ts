import { QueryClient } from '@tanstack/react-query';

import { getGetProjectConfigQueryKey } from '@ui/gen/api/v2/core/core';
import { CreateOrUpdateResourceResult } from '@ui/gen/api/v2/models';
import { getGetClusterConfigQueryKey } from '@ui/gen/api/v2/system/system';

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
