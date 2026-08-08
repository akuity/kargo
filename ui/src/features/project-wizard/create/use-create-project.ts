import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useState } from 'react';

import { getListProjectsQueryKey } from '@ui/gen/api/v2/core/core';
import { createResource } from '@ui/gen/api/v2/resources/resources';

import { creationManifests } from '../manifest/manifest-builder';
import { WizardState } from '../types';

import { ProgressItem, mergeForRetry, runCreate, toProgressItems } from './create-engine';

export type CreateStatus = 'idle' | 'running' | 'error' | 'done';

const createFn = (manifestYaml: string) => createResource(manifestYaml).then(() => undefined);

export const useCreateProject = (state: WizardState) => {
  const queryClient = useQueryClient();
  const [items, setItems] = useState<ProgressItem[]>([]);
  const [status, setStatus] = useState<CreateStatus>('idle');

  const execute = useCallback(
    async (progressItems: ProgressItem[]) => {
      setStatus('running');
      const ok = await runCreate(progressItems, createFn, setItems);
      if (ok) {
        queryClient.invalidateQueries({ queryKey: getListProjectsQueryKey() });
      }
      setStatus(ok ? 'done' : 'error');
    },
    [queryClient]
  );

  // Fresh run from the current wizard state.
  const run = useCallback(
    () => execute(toProgressItems(creationManifests(state))),
    [execute, state]
  );

  // Resume after a failure: regenerate manifests from the (possibly edited)
  // current state so fixes like a renamed project take effect, then freeze any
  // resource this run already created so it is not re-applied (see
  // mergeForRetry) -- this avoids "already exists" errors on retry.
  const retry = useCallback(() => {
    execute(mergeForRetry(items, toProgressItems(creationManifests(state))));
  }, [execute, items, state]);

  return { items, status, run, retry };
};
