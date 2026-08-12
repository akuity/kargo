import { MutationStatus, useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

import {
  StageConditionStatus,
  StageConditionType,
  hasCondition
} from '@ui/features/common/stage-status/utils';
import { getListProjectsQueryKey, getProject } from '@ui/gen/api/v2/core/core';
import { V1Condition } from '@ui/gen/api/v2/models';
import { createResource } from '@ui/gen/api/v2/resources/resources';

import { creationManifests } from '../manifest/manifest-builder';
import { WizardState } from '../types';

import { ProgressItem, mergeForRetry, runCreate, toProgressItems } from './create-engine';

export type CreateStatus = MutationStatus;

const createFn = (manifestYaml: string) => createResource(manifestYaml).then(() => undefined);

const READY_TIMEOUT_MS = 30_000;
const READY_POLL_INTERVAL_MS = 1_000;

const isReady = (conditions: V1Condition[] = []) =>
  hasCondition(StageConditionType.Ready, StageConditionStatus.True, conditions).isActive;

// Everything after the Project lands in the Namespace the Project provisions
// asynchronously, so gate on the Project reporting Ready.
const awaitProjectReady = (item: ProgressItem): Promise<void> | undefined => {
  if (item.kind !== 'Project') {
    return undefined;
  }
  return (async () => {
    const deadline = Date.now() + READY_TIMEOUT_MS;
    for (;;) {
      let conditions: V1Condition[] | undefined;
      try {
        conditions = (await getProject(item.name)).data?.status?.conditions;
      } catch {
        // Reads can fail while the Project settles; the deadline ends the loop.
      }
      if (isReady(conditions)) {
        return;
      }
      if (Date.now() >= deadline) {
        throw new Error(
          `Project "${item.name}" was created but did not report ready within ` +
            `${READY_TIMEOUT_MS / 1000}s. Its namespace may still be provisioning — retry to resume.`
        );
      }
      await new Promise((resolve) => setTimeout(resolve, READY_POLL_INTERVAL_MS));
    }
  })();
};

export const useCreateProject = (state: WizardState) => {
  const queryClient = useQueryClient();
  const [items, setItems] = useState<ProgressItem[]>([]);

  const { status, mutate } = useMutation({
    mutationFn: async (progressItems: ProgressItem[]) => {
      const ok = await runCreate(progressItems, createFn, setItems, {
        awaitReady: awaitProjectReady
      });
      // Rejecting is what marks the mutation errored. A plain Error (not an
      // ApiError) keeps config/query-client from also firing a toast.
      if (!ok) {
        throw new Error('creation halted');
      }
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: getListProjectsQueryKey() })
  });

  return {
    items,
    status,
    run: () => mutate(toProgressItems(creationManifests(state))),
    // Regenerate from current state so edits take effect, then freeze what this
    // run already created (see mergeForRetry) to avoid "already exists".
    retry: () => mutate(mergeForRetry(items, toProgressItems(creationManifests(state))))
  };
};
