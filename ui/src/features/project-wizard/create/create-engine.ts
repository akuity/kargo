import { apiErrorBodyMessage, isApiErrorLike } from '@ui/lib/api/error-message';

import { CreationManifest } from '../manifest/manifest-builder';

export type ItemState = 'pending' | 'running' | 'done' | 'error';

export type ProgressItem = CreationManifest & {
  state: ItemState;
  message?: string;
  // Set once createFn has returned successfully. Tracked apart from the 'done'
  // state because an item can be created and still fail its readiness gate: a
  // retry must skip the create (which would now collide) but re-run the gate.
  created?: boolean;
};

export type CreateFn = (manifestYaml: string) => Promise<void>;

type RunOptions = {
  sleep?: (ms: number) => Promise<void>;
  // Pause while a step shows as "running" so fast creates are still noticable
  stepDelayMs?: number;
  // Awaited after an item is created and before the next one starts, so later
  // resources don't race state the API settles asynchronously -- the Namespace
  // a Project provisions, say. Returns undefined for items that need no gate.
  awaitReady?: (item: ProgressItem) => Promise<void> | undefined;
};

export const toProgressItems = (manifests: CreationManifest[]): ProgressItem[] =>
  manifests.map((m) => ({ ...m, state: 'pending' }));

// Freezes resources this run already created so a retry never re-applies them.
// A prior success is matched to a freshly generated manifest by kind+name only
// -- the manifest body is intentionally ignored, so edits to an already-created
// resource are NOT re-applied on retry (a create-only flow can't mutate what it
// already created; edit it from its own page instead). This keeps genuine
// "already exists" collisions -- e.g. a Project name taken by someone else,
// which was never created here -- failing loudly, while newly added, failed, or
// still-pending resources are re-run. A resource that was created but never
// passed its readiness gate carries `created` forward instead, so the retry
// re-runs only the gate.
export const mergeForRetry = (previous: ProgressItem[], fresh: ProgressItem[]): ProgressItem[] => {
  const nameKey = (i: CreationManifest) => `${i.kind}/${i.name}`;
  const priorByKey = new Map(previous.map((i) => [nameKey(i), i]));
  return fresh.map((i) => {
    const prior = priorByKey.get(nameKey(i));
    if (prior?.state === 'done') {
      return { ...i, state: 'done' as const, message: 'Created', created: true };
    }
    return prior?.created ? { ...i, created: true } : i;
  });
};

// Progress rows are already labelled with the resource they belong to, so the
// bare statusText reads better here than ApiError's "API Error: 404 Not Found".
export const errorMessage = (error: unknown): string => {
  if (isApiErrorLike(error)) {
    return (
      apiErrorBodyMessage(error.body) || error.statusText || error.message || `HTTP ${error.status}`
    );
  }
  return error instanceof Error ? error.message : String(error);
};

// Sequentially applies each pending item via createFn, emitting progress after
// every state change. On failure it marks the item errored, flags the rest as
// halted, and returns false (leaving done items intact so a retry resumes).
// Already-done items are skipped, which makes this reusable as a retry.
export const runCreate = async (
  items: ProgressItem[],
  createFn: CreateFn,
  onProgress: (items: ProgressItem[]) => void,
  options: RunOptions = {}
): Promise<boolean> => {
  const stepDelayMs = options.stepDelayMs ?? 200;
  const sleep = options.sleep ?? ((ms) => new Promise((resolve) => setTimeout(resolve, ms)));

  const next = items.map((i) => ({ ...i }));
  const emit = () => onProgress(next.map((i) => ({ ...i })));

  // Mark the item failed and flag everything after it as halted.
  const halt = (index: number, error: unknown) => {
    next[index] = { ...next[index], state: 'error', message: errorMessage(error) };
    for (let j = index + 1; j < next.length; j++) {
      next[j] = { ...next[j], state: 'pending', message: 'Halted by prior failure' };
    }
    emit();
  };

  for (let i = 0; i < next.length; i++) {
    if (next[i].state === 'done') {
      continue;
    }

    next[i] = { ...next[i], state: 'running', message: undefined };
    emit();
    await sleep(stepDelayMs);

    // A resource an earlier attempt created but left un-ready is not re-applied
    // (that would collide); only its gate is re-run.
    if (!next[i].created) {
      try {
        await createFn(next[i].yaml);
      } catch (error) {
        halt(i, error);
        return false;
      }
      next[i] = { ...next[i], created: true };
    }

    const ready = options.awaitReady?.(next[i]);
    if (ready) {
      next[i] = { ...next[i], message: 'Waiting until ready' };
      emit();
      try {
        await ready;
      } catch (error) {
        halt(i, error);
        return false;
      }
    }

    next[i] = { ...next[i], state: 'done', message: 'Created' };
    emit();
  }

  emit();
  return true;
};
