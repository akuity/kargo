import { useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useRef } from 'react';

import {
  getGetTargetStatusSummaryQueryKey,
  getListTargetStatusesQueryKey,
  getTargetStatusSummary,
  listTargetStatuses
} from '@ui/gen/api/v2/core/core';
import { Stage, Target, TargetStatusEntry, TargetStatusSummary } from '@ui/gen/api/v2/models';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { orEmpty, polled } from './database-reads';
import {
  shouldPollList,
  shouldPollSummary,
  statusSignature,
  targetsSignature
} from './model/target-status';

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

// useRefetchOnChange reads statuses again whenever signature changes: a
// round starts or ends, a Stage takes on new Freight or new selectors, or a
// Target comes, goes or is relabelled. Polling stops when nothing is in
// flight, so this is what picks up what happens next. Until ready, signature
// is not yet worth comparing -- what it is read from is still loading -- and
// the first one seen once ready is where comparison starts, not a change.
const useRefetchOnChange = (signature: string, ready: boolean, refetch: () => unknown) => {
  const seen = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (!ready) {
      return;
    }
    if (seen.current !== undefined && seen.current !== signature) {
      refetch();
    }
    seen.current = signature;
  }, [signature, ready, refetch]);
};

// useTargetStatuses reads where each Target stands with each Stage that
// governs it, for a whole Project or, given a Stage name, for that Stage. It
// is read again while a promotion is in flight to any of them, or a Stage
// says it is promoting, and once more whenever the Stages or the Targets move
// on. targets is the list the page shows, or undefined while it loads.
export const useTargetStatuses = (
  project: string,
  stages: Stage[],
  targets: Target[] | undefined,
  stage?: string
) => {
  const enabled = !!project;
  const params = stage ? { stage } : undefined;

  const listQuery = useQuery({
    queryKey: [...getListTargetStatusesQueryKey(project, params), 'all'],
    queryFn: () => orEmpty(() => readAllStatuses(project, stage), []),
    enabled,
    meta: polled,
    refetchInterval: (query) =>
      shouldPollList(
        { failed: query.state.status === 'error', updatedAt: query.state.dataUpdatedAt },
        query.state.data || [],
        stages
      )
        ? TARGETS_POLL_INTERVAL_MS
        : false
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
    // The summary moves with the list, so it is polled by the list's rule,
    // and once more after the list's latest read.
    refetchInterval: (query) =>
      shouldPollSummary(
        { failed: query.state.status === 'error', updatedAt: query.state.dataUpdatedAt },
        { failed: listQuery.isError, updatedAt: listQuery.dataUpdatedAt },
        listQuery.data || [],
        stages
      )
        ? TARGETS_POLL_INTERVAL_MS
        : false
  });

  const refetch = listQuery.refetch;
  const refetchSummary = summaryQuery.refetch;
  const refetchBoth = useMemo(
    () => () => Promise.all([refetch(), refetchSummary()]),
    [refetch, refetchSummary]
  );
  useRefetchOnChange(statusSignature(stages), enabled, refetchBoth);
  const targetsKey = useMemo(() => (targets ? targetsSignature(targets) : ''), [targets]);
  useRefetchOnChange(targetsKey, enabled && !!targets, refetchBoth);

  return {
    entries: listQuery.data,
    summary: summaryQuery.data,
    isLoading: enabled && (listQuery.isLoading || summaryQuery.isLoading),
    // Set only while no read of the list has succeeded, so there are no
    // statuses to show. That is not an empty fleet, and must not be drawn as
    // one. A read that fails after one has succeeded leaves its entries in
    // place; the failure is reported once, and the list is read again.
    error: listQuery.data ? null : listQuery.error
  };
};
