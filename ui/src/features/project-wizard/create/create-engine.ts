import { apiErrorBodyMessage, isApiErrorLike } from '@ui/lib/api/error-message';

import { CreationManifest } from '../manifest/manifest-builder';

export type ItemState = 'pending' | 'running' | 'done' | 'error';

export type ProgressItem = CreationManifest & {
  state: ItemState;
  message?: string;
};

export type CreateFn = (manifestYaml: string) => Promise<void>;

type RunOptions = {
  retries?: number;
  delayMs?: number;
  isRetryable?: (error: unknown) => boolean;
  sleep?: (ms: number) => Promise<void>;
  // Pause while a step shows as "running" so fast creates are still noticable
  stepDelayMs?: number;
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
// still-pending resources are re-run.
export const mergeForRetry = (previous: ProgressItem[], fresh: ProgressItem[]): ProgressItem[] => {
  const nameKey = (i: CreationManifest) => `${i.kind}/${i.name}`;
  const created = new Set(previous.filter((i) => i.state === 'done').map(nameKey));
  return fresh.map((i) =>
    created.has(nameKey(i)) ? { ...i, state: 'done' as const, message: 'Created' } : i
  );
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

// After a Project is created, its Namespace is provisioned asynchronously, so
// the next resource create can briefly fail with "namespace not found". Only
// those transient errors are worth retrying.
export const isNamespaceNotReady = (error: unknown): boolean =>
  isApiErrorLike(error) && /namespace.*not found|not found.*namespace/i.test(errorMessage(error));

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
  const retries = options.retries ?? 7;
  const delayMs = options.delayMs ?? 1500;
  const stepDelayMs = options.stepDelayMs ?? 200;
  const isRetryable = options.isRetryable ?? isNamespaceNotReady;
  const sleep = options.sleep ?? ((ms) => new Promise((resolve) => setTimeout(resolve, ms)));

  const next = items.map((i) => ({ ...i }));
  const emit = () => onProgress(next.map((i) => ({ ...i })));

  for (let i = 0; i < next.length; i++) {
    if (next[i].state === 'done') {
      continue;
    }

    next[i] = { ...next[i], state: 'running', message: undefined };
    emit();
    await sleep(stepDelayMs);

    let lastError: unknown;
    for (let attempt = 0; attempt <= retries; attempt++) {
      try {
        await createFn(next[i].yaml);
        lastError = undefined;
        break;
      } catch (error) {
        lastError = error;
        if (attempt < retries && isRetryable(error)) {
          await sleep(delayMs);
          continue;
        }
        break;
      }
    }

    if (lastError) {
      next[i] = { ...next[i], state: 'error', message: errorMessage(lastError) };
      for (let j = i + 1; j < next.length; j++) {
        next[j] = { ...next[j], state: 'pending', message: 'Halted by prior failure' };
      }
      emit();
      return false;
    }

    next[i] = { ...next[i], state: 'done', message: 'Created' };
    emit();
  }

  emit();
  return true;
};
