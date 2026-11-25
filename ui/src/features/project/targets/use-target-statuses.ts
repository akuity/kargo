import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef } from 'react';

import {
  getGetTargetStatusSummaryQueryKey,
  getListTargetStatusesQueryKey,
  getTargetStatusSummary,
  listTargetStatuses
} from '@ui/gen/api/v2/core/core';
import { Stage, Target, TargetStatusEntry, TargetStatusSummary } from '@ui/gen/api/v2/models';

import { TARGETS_POLL_INTERVAL_MS } from './constants';
import { orEmpty, polled, readFresh } from './database-reads';
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
// flight, so this is what picks up what happens next. The signature when the
// statuses are first read is where comparison starts, not a change.
const useRefetchOnChange = (signature: string, ready: boolean, readAgain: () => unknown) => {
  const seen = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (!ready) {
      return;
    }
    if (seen.current !== undefined && seen.current !== signature) {
      readAgain();
    }
    seen.current = signature;
  }, [signature, ready, readAgain]);
};

// useTargetStatuses reads where each Target stands with each Stage that
// governs it, for a whole Project or, given a Stage name, for that Stage. It
// is read again while a promotion is in flight to any of them, or a Stage
// says it is promoting, and once more whenever the Stages or the Targets move
// on.
//
// stages and targets are the lists the page shows, undefined while they load.
// The statuses are first read once both have: read any sooner, they could
// predate a Target the list then shows, and nothing would read them again
// until something else changed.
export const useTargetStatuses = (
  project: string,
  stages: Stage[] | undefined,
  targets: Target[] | undefined,
  stage?: string
) => {
  const ready = !!project && !!stages && !!targets;
  const shownStages = useMemo(() => stages || [], [stages]);
  const listKey = useMemo(
    () => [...getListTargetStatusesQueryKey(project, stage ? { stage } : undefined), 'all'],
    [project, stage]
  );
  const summaryKey = useMemo(
    () => getGetTargetStatusSummaryQueryKey(project, stage ? { stage } : undefined),
    [project, stage]
  );

  const listQuery = useQuery({
    queryKey: listKey,
    queryFn: () => orEmpty(() => readAllStatuses(project, stage), []),
    enabled: ready,
    meta: polled,
    refetchInterval: (query) =>
      shouldPollList(
        { failed: query.state.status === 'error', updatedAt: query.state.dataUpdatedAt },
        query.state.data || [],
        shownStages
      )
        ? TARGETS_POLL_INTERVAL_MS
        : false
  });

  const summaryQuery = useQuery({
    queryKey: summaryKey,
    queryFn: () =>
      orEmpty(
        async () => (await getTargetStatusSummary(project, stage ? { stage } : undefined)).data,
        {} as TargetStatusSummary
      ),
    enabled: ready,
    meta: polled,
    // The summary moves with the list, so it is polled by the list's rule,
    // and once more after the list's latest read.
    refetchInterval: (query) =>
      shouldPollSummary(
        { failed: query.state.status === 'error', updatedAt: query.state.dataUpdatedAt },
        { failed: listQuery.isError, updatedAt: listQuery.dataUpdatedAt },
        listQuery.data || [],
        shownStages
      )
        ? TARGETS_POLL_INTERVAL_MS
        : false
  });

  const queryClient = useQueryClient();
  const readAgain = useCallback(
    () => Promise.all([readFresh(queryClient, listKey), readFresh(queryClient, summaryKey)]),
    [queryClient, listKey, summaryKey]
  );
  useRefetchOnChange(statusSignature(shownStages), ready, readAgain);
  const targetsKey = useMemo(() => targetsSignature(targets || []), [targets]);
  useRefetchOnChange(targetsKey, ready, readAgain);

  return {
    entries: listQuery.data,
    summary: summaryQuery.data,
    // Loading, too, while what the statuses wait on loads.
    isLoading: !!project && (!ready || listQuery.isLoading || summaryQuery.isLoading),
    // Set only while no read of the list has succeeded, so there are no
    // statuses to show. That is not an empty fleet, and must not be drawn as
    // one. A read that fails after one has succeeded leaves its entries in
    // place; the failure is reported once, and the list is read again.
    error: listQuery.data ? null : listQuery.error
  };
};
