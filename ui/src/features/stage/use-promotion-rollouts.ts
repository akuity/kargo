import { useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef } from 'react';

import {
  StagePromotionRef,
  getCurrentPromotionRef,
  isStageTargetAware
} from '@ui/features/project/pipelines/nodes/stage-meta-utils';
import {
  getListPromotionRolloutsQueryKey,
  listPromotionRollouts,
  listPromotionRolloutsResponse
} from '@ui/gen/api/v2/core/core';
import { ListPromotionRolloutsParams, PromotionRollout, Stage } from '@ui/gen/api/v2/models';
import { ApiError } from '@ui/lib/api/custom-fetch';

import {
  LatestRollouts,
  latestRolloutsByStage,
  recordedRollouts,
  rolloutCompareFn,
  shouldPollRollouts
} from './utils/promotion-rollout';

// PromotionRollouts live in the database, which offers no watch, so a list of
// them is read again at this interval for as long as it can still change.
export const ROLLOUTS_POLL_INTERVAL_MS = 5_000;

const notImplemented = 501;

// listRollouts reads one page of PromotionRollouts. A server that runs
// without a database serves none and says so with a 501; to a reader that is
// a Project with no rollouts, not a failure to report, so it is answered as
// an empty list.
const listRollouts = async (
  project: string,
  params: ListPromotionRolloutsParams
): Promise<listPromotionRolloutsResponse> => {
  try {
    return await listPromotionRollouts(project, params);
  } catch (error) {
    if (error instanceof ApiError && error.is(notImplemented)) {
      return { data: { items: [] }, status: 200, headers: new Headers() };
    }
    throw error;
  }
};

// pollInterval is the refetchInterval of a rollout list: the poll interval
// while the Stages reading it have something in flight, and no polling
// otherwise. A list that fails to read is not polled either -- left on the
// timer it would raise the global error notification on every tick.
const pollInterval = (failed: boolean, stages: Stage[], rollouts: PromotionRollout[]) =>
  !failed && shouldPollRollouts(stages, rollouts) ? ROLLOUTS_POLL_INTERVAL_MS : false;

// useRefetchOnRecorded reads a rollout list again whenever one of the Stages
// records a rollout it had not recorded before. Polling stops when nothing is
// in flight, so this is what picks up a rollout that came and went unseen.
const useRefetchOnRecorded = (stages: Stage[], enabled: boolean, refetch: () => unknown) => {
  const recorded = recordedRollouts(stages);
  const seen = useRef(recorded);
  useEffect(() => {
    if (seen.current === recorded) {
      return;
    }
    seen.current = recorded;
    if (enabled) {
      refetch();
    }
  }, [recorded, enabled, refetch]);
};

const latestParams: ListPromotionRolloutsParams = { latest: true };

// useLatestRollouts reads where each of a Project's Stages stands with its
// PromotionRollouts -- the one it is promoting through and the newest that
// finished -- keyed by Stage name. Stages name no current rollout, so every
// view of a rollout in flight is drawn from this.
//
// It is one list for the whole Project, under one query key, so a pipeline of
// Stage nodes, a Stage's drawer and the Targets page all share a single read.
// Nothing is read for a Project with no Stage that governs Targets.
export const useLatestRollouts = (
  project: string,
  stages: Stage[]
): { byStage: Record<string, LatestRollouts>; isLoading: boolean } => {
  const fleet = useMemo(() => stages.filter(isStageTargetAware), [stages]);
  const enabled = !!project && fleet.length > 0;

  const query = useQuery({
    queryKey: getListPromotionRolloutsQueryKey(project, latestParams),
    queryFn: () => listRollouts(project, latestParams),
    enabled,
    refetchInterval: (current) =>
      pollInterval(current.state.status === 'error', fleet, current.state.data?.data?.items || [])
  });
  useRefetchOnRecorded(fleet, enabled, query.refetch);

  const byStage = useMemo(() => latestRolloutsByStage(query.data?.data?.items || []), [query.data]);
  return { byStage, isLoading: enabled && query.isLoading };
};

// useStageLatestRollouts is useLatestRollouts for one Stage. It reads the
// same Project-wide list, so asking per Stage costs no further requests.
export const useStageLatestRollouts = (
  project: string,
  stage?: Stage
): { latest?: LatestRollouts; isLoading: boolean } => {
  const stages = useMemo(() => (stage ? [stage] : []), [stage]);
  const { byStage, isLoading } = useLatestRollouts(project, stages);
  return { latest: byStage[stage?.metadata?.name || ''], isLoading };
};

// useCurrentPromotionRef describes what a Stage is promoting right now, if
// anything, for the surfaces that say so in passing: a pipeline node, a list
// row, a Stage's tag. It is getCurrentPromotionRef with the Stage's rollout in
// flight filled in from the list, which the Stage itself does not name.
export const useCurrentPromotionRef = (stage?: Stage): StagePromotionRef | undefined => {
  const { latest } = useStageLatestRollouts(stage?.metadata?.namespace || '', stage);
  return getCurrentPromotionRef(stage, latest?.unfinished);
};

// useStageRollouts reads a Stage's PromotionRollouts, newest first: its
// history. The server answers one page at a time and names where the next
// starts in metadata.continue; loadMore reads on from there. The pages read so
// far are polled while the Stage has a rollout in flight.
export const useStageRollouts = (project: string, stage?: Stage) => {
  const stageName = stage?.metadata?.name || '';
  const stages = useMemo(() => (stage ? [stage] : []), [stage]);
  const enabled = !!project && !!stageName;

  const query = useInfiniteQuery({
    queryKey: [...getListPromotionRolloutsQueryKey(project, { stage: stageName }), 'pages'],
    queryFn: ({ pageParam }) =>
      listRollouts(project, {
        stage: stageName,
        ...(pageParam ? { continue: pageParam } : {})
      }),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.data?.metadata?.continue || undefined,
    enabled,
    refetchInterval: (current) =>
      pollInterval(
        current.state.status === 'error',
        stages,
        (current.state.data?.pages || []).flatMap((page) => page.data?.items || [])
      )
  });
  useRefetchOnRecorded(stages, enabled, query.refetch);

  // A rollout created between the reads of two pages moves every older one
  // down a place, so the same rollout can close one page and open the next.
  const rollouts = useMemo(() => {
    const byName = new Map<string, PromotionRollout>();
    for (const page of query.data?.pages || []) {
      for (const rollout of page.data?.items || []) {
        byName.set(rollout.metadata?.name || '', rollout);
      }
    }
    return [...byName.values()].sort(rolloutCompareFn);
  }, [query.data]);

  return {
    rollouts,
    isLoading: enabled && query.isLoading,
    hasMore: !!query.hasNextPage,
    isLoadingMore: query.isFetchingNextPage,
    loadMore: query.fetchNextPage
  };
};
