import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef } from 'react';

import {
  getGetTargetStatusSummaryQueryKey,
  getListTargetStatusesQueryKey,
  getTargetStatusSummary,
  listTargetStatuses
} from '@ui/gen/api/v2/core/core';
import { Stage, TargetStatusEntry, TargetStatusSummary } from '@ui/gen/api/v2/models';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { orEmpty, polled } from './database-reads';
import { shouldPollStatuses, statusSignature } from './model/target-status';

// The most entries the server returns in one page.
const pageSize = 500;

// readAllStatuses reads every page of a Project's Target statuses, or of one
// Stage's. The server names where the next page starts; a fleet of thousands
// is several pages.
const readAllStatuses = async (project: string, stage?: string): Promise<TargetStatusEntry[]> => {
  const entries: TargetStatusEntry[] = [];
  let next: string | undefined;
  do {
    const response = await listTargetStatuses(project, {
      ...(stage ? { stage } : {}),
      limit: pageSize,
      ...(next ? { continue: next } : {})
    });
    entries.push(...(response.data?.items || []));
    next = response.data?.continue || undefined;
  } while (next);
  return entries;
};

// useRefetchOnChange reads statuses again whenever the Stages move on: a
// round starts or ends, or a Stage takes on new Freight. Polling stops when
// nothing is in flight, so this is what picks up what happens next.
const useRefetchOnChange = (stages: Stage[], enabled: boolean, refetch: () => unknown) => {
  const signature = statusSignature(stages);
  const seen = useRef(signature);
  useEffect(() => {
    if (seen.current === signature) {
      return;
    }
    seen.current = signature;
    if (enabled) {
      refetch();
    }
  }, [signature, enabled, refetch]);
};

// useTargetStatuses reads where each Target stands with each Stage that
// governs it, for a whole Project or, given a Stage name, for that Stage. It
// is read again while a promotion is in flight to any of them, or a Stage
// says it is promoting, and once more whenever the Stages move on.
export const useTargetStatuses = (project: string, stages: Stage[], stage?: string) => {
  const enabled = !!project;
  const params = stage ? { stage } : undefined;

  const listQuery = useQuery({
    queryKey: [...getListTargetStatusesQueryKey(project, params), 'all'],
    queryFn: () => orEmpty(() => readAllStatuses(project, stage), []),
    enabled,
    meta: polled,
    refetchInterval: (query) =>
      shouldPollStatuses(query.state.data || [], stages) ? TARGETS_POLL_INTERVAL_MS : false
  });

  const summaryQuery = useQuery({
    queryKey: getGetTargetStatusSummaryQueryKey(project, params),
    queryFn: () =>
      orEmpty(
        async () => (await getTargetStatusSummary(project, params)).data,
        {} as TargetStatusSummary
      ),
    enabled,
    meta: polled,
    // The summary moves with the list, so it is polled by the same rule.
    refetchInterval: () =>
      shouldPollStatuses(listQuery.data || [], stages) ? TARGETS_POLL_INTERVAL_MS : false
  });

  const refetch = listQuery.refetch;
  const refetchSummary = summaryQuery.refetch;
  const refetchBoth = useMemo(
    () => () => Promise.all([refetch(), refetchSummary()]),
    [refetch, refetchSummary]
  );
  useRefetchOnChange(stages, enabled, refetchBoth);

  return {
    entries: listQuery.data,
    summary: summaryQuery.data,
    isLoading: enabled && (listQuery.isLoading || summaryQuery.isLoading)
  };
};
