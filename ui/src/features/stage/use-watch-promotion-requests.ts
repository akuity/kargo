import { QueryClient, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import {
  getListPromotionRequestsQueryKey,
  listPromotionRequestsResponse
} from '@ui/gen/api/v2/core/core';
import { PromotionRequest } from '@ui/gen/api/v2/models';

import { runSeededWatch, upsertOrDelete } from '../project/pipelines/watch-utils';

// A watch writes into one query's cache, so any number of components reading
// the same list need only one. Watches are shared per query client and list
// key, counted by subscriber, and closed when the last one unmounts: the Stage
// drawer's current round and its Promotions tab read the same list, and
// without this each opened a stream of its own.
type SharedWatch = { subscribers: number; abort: AbortController };
const sharedWatches = new WeakMap<QueryClient, Map<string, SharedWatch>>();

export const shareWatch = (
  client: QueryClient,
  key: string,
  start: (signal: AbortSignal) => void
) => {
  let watches = sharedWatches.get(client);
  if (!watches) {
    watches = new Map();
    sharedWatches.set(client, watches);
  }
  let watch = watches.get(key);
  if (!watch) {
    watch = { subscribers: 0, abort: new AbortController() };
    watches.set(key, watch);
    start(watch.abort.signal);
  }
  watch.subscribers++;
  const shared = watch;
  const all = watches;
  return () => {
    shared.subscribers--;
    if (shared.subscribers === 0) {
      shared.abort.abort();
      all.delete(key);
    }
  };
};

// useWatchPromotionRequests keeps a PromotionRequest list current. With a
// stage, it follows that Stage's requests; without one, every request in the
// project. It updates the query issued with the same params, since they form
// the query key.
//
// enabled gates the watch on the initial list having loaded, so it never opens
// before a seed resourceVersion is available — otherwise it would start an
// unseeded watch that replays every PromotionRequest.
export const useWatchPromotionRequests = (project: string, stage?: string, enabled = true) => {
  const client = useQueryClient();

  useEffect(() => {
    // An empty stage is a caller that meant to scope the watch and had no name
    // to scope it by. Watching the whole project instead would write into a
    // cache key that caller is not reading.
    if (!project || !enabled || stage === '') {
      return;
    }

    const listKey = getListPromotionRequestsQueryKey(project, stage ? { stage } : undefined);

    const seedResourceVersion = () =>
      (client.getQueryData(listKey) as listPromotionRequestsResponse | undefined)?.data?.metadata
        ?.resourceVersion;

    const buildUrl = (resourceVersion: string) => {
      const params = new URLSearchParams({ watch: 'true' });
      if (stage) {
        params.append('stage', stage);
      }
      if (resourceVersion) {
        params.append('resourceVersion', resourceVersion);
      }
      return `/v1beta1/projects/${encodeURIComponent(project)}/promotion-requests?${params}`;
    };

    const relist = async () => {
      await client.refetchQueries({ queryKey: listKey, exact: false });
      return seedResourceVersion();
    };

    const onEvent = (type: string, promotionRequest: PromotionRequest) => {
      client.setQueryData(listKey, (old: listPromotionRequestsResponse | undefined) => {
        if (!old?.data) {
          return old;
        }
        return {
          ...old,
          data: {
            ...old.data,
            items: upsertOrDelete(old.data.items ?? [], promotionRequest, type)
          }
        };
      });
    };

    return shareWatch(client, JSON.stringify(listKey), (signal) =>
      runSeededWatch<PromotionRequest>({
        signal,
        buildUrl,
        seedResourceVersion,
        relist,
        onEvent
      })
    );
  }, [project, stage, client, enabled]);
};
