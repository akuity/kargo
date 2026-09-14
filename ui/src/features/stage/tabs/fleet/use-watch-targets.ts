import { useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { runSeededWatch, upsertOrDelete } from '@ui/features/project/pipelines/watch-utils';
import { getListTargetsQueryKey, listTargetsResponse } from '@ui/gen/api/v2/core/core';
import { Target } from '@ui/gen/api/v2/models';

// useWatchTargets keeps the list of Targets a Stage governs current. It
// updates the query issued with the same `stage` param, since that forms the
// query key. As with the other collection watches, `enabled` gates it on the
// initial list having loaded so the watch is always seeded with a
// resourceVersion.
export const useWatchTargets = (project: string, stage: string, enabled = true) => {
  const client = useQueryClient();

  useEffect(() => {
    if (!project || !stage || !enabled) {
      return;
    }

    const abort = new AbortController();
    const listKey = getListTargetsQueryKey(project, { stage });

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listTargetsResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true', stage });
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/targets?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, target: Target) => {
      client.setQueryData(listKey, (old: listTargetsResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: { ...old.data, items: upsertOrDelete(old.data.items ?? [], target, type) }
        };
      });
    };

    runSeededWatch<Target>({
      signal: abort.signal,
      buildUrl,
      seedResourceVersion,
      relist,
      onEvent
    });

    return () => abort.abort();
  }, [project, stage, client, enabled]);
};
