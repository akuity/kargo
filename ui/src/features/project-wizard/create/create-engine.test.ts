import { expect, test } from 'vitest';

import {
  ProgressItem,
  createdProjectName,
  errorMessage,
  mergeForRetry,
  runCreate,
  toProgressItems
} from './create-engine';

// Mirrors the REST client's ApiError shape (status/statusText/body) that the
// engine duck-types, without importing the HTTP client.
const apiError = (status: number, statusText: string, body: unknown) => ({
  status,
  statusText,
  body,
  message: `API Error: ${status} ${statusText}`
});

const items = (): ProgressItem[] =>
  toProgressItems([
    { kind: 'Project', name: 'p', yaml: 'a' },
    { kind: 'Secret', name: 's', yaml: 'b' },
    { kind: 'Warehouse', name: 'w', yaml: 'c' }
  ]);

const noSleep = () => Promise.resolve();

test('runCreate applies every item in order and reports done', async () => {
  const applied: string[] = [];
  const states: string[][] = [];
  const ok = await runCreate(
    items(),
    async (y) => {
      applied.push(y);
    },
    (progress) => states.push(progress.map((i) => i.state)),
    { sleep: noSleep }
  );
  expect(ok).toBe(true);
  expect(applied).toEqual(['a', 'b', 'c']);
  // final emit: everything done
  expect(states.at(-1)).toEqual(['done', 'done', 'done']);
});

test('runCreate halts remaining items on failure and returns false', async () => {
  const ok = await runCreate(
    items(),
    async (y) => {
      if (y === 'b') {
        throw apiError(422, 'Unprocessable', 'bad secret');
      }
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(false);
});

test('runCreate marks the failed item errored and the rest halted', async () => {
  let final: ProgressItem[] = [];
  await runCreate(
    items(),
    async (y) => {
      if (y === 'b') {
        throw apiError(422, 'Unprocessable', 'bad secret');
      }
    },
    (progress) => {
      final = progress;
    },
    { sleep: noSleep }
  );
  expect(final.map((i) => i.state)).toEqual(['done', 'error', 'pending']);
  expect(final[1].message).toBe('bad secret');
  expect(final[2].message).toBe('Halted by prior failure');
});

// A failed create is applied once and halts the run. Transient-failure handling
// belongs on the mutation, not in a hand-rolled loop here.
test('runCreate applies a failing create exactly once', async () => {
  let attempts = 0;
  const ok = await runCreate(
    toProgressItems([{ kind: 'Secret', name: 's', yaml: 'b' }]),
    async () => {
      attempts++;
      throw apiError(503, 'Service Unavailable', 'try again');
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(false);
  expect(attempts).toBe(1);
});

test('runCreate resumes from a failed item, skipping done ones (retry)', async () => {
  const applied: string[] = [];
  // first pass: 'b' fails
  const progress = items();
  let captured: ProgressItem[] = [];
  await runCreate(
    progress,
    async (y) => {
      applied.push(y);
      if (y === 'b') {
        throw apiError(422, 'x', 'boom');
      }
    },
    (p) => {
      captured = p;
    },
    { sleep: noSleep }
  );
  expect(applied).toEqual(['a', 'b']);

  // retry: re-run the captured progress with a now-working createFn
  applied.length = 0;
  const ok = await runCreate(
    captured,
    async (y) => {
      applied.push(y);
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(true);
  // 'a' was already done and is skipped; only 'b' and 'c' are applied
  expect(applied).toEqual(['b', 'c']);
});

test('mergeForRetry freezes created resources and re-runs the rest', () => {
  const previous: ProgressItem[] = [
    { kind: 'Project', name: 'p', yaml: 'a', state: 'done', message: 'Created' },
    { kind: 'Secret', name: 's', yaml: 'b', state: 'error', message: 'boom' }
  ];
  // fresh adds a Warehouse the previous run never reached
  const fresh = toProgressItems([
    { kind: 'Project', name: 'p', yaml: 'a' },
    { kind: 'Secret', name: 's', yaml: 'b' },
    { kind: 'Warehouse', name: 'w', yaml: 'c' }
  ]);
  const merged = mergeForRetry(previous, fresh);
  expect(merged.map((i) => [i.kind, i.state])).toEqual([
    ['Project', 'done'],
    ['Secret', 'pending'],
    ['Warehouse', 'pending']
  ]);
  expect(merged[0].message).toBe('Created');
});

test('a created resource is not re-applied on retry even after editing it', async () => {
  const previous: ProgressItem[] = [
    { kind: 'Project', name: 'p', yaml: 'old', state: 'done', message: 'Created' },
    { kind: 'Warehouse', name: 'w', yaml: 'c', state: 'error', message: 'boom' }
  ];
  // the user edited the already-created Project (yaml changed) and fixed the Warehouse
  const fresh = toProgressItems([
    { kind: 'Project', name: 'p', yaml: 'edited' },
    { kind: 'Warehouse', name: 'w', yaml: 'c2' }
  ]);
  const applied: string[] = [];
  const ok = await runCreate(
    mergeForRetry(previous, fresh),
    async (y) => {
      applied.push(y);
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(true);
  // Project is frozen (never re-applied, so no "already exists"); only the
  // previously-failed Warehouse is applied.
  expect(applied).toEqual(['c2']);
});

test('runCreate gates the next item on awaitReady', async () => {
  const events: string[] = [];
  let releaseProject: () => void = () => {};
  const projectReady = new Promise<void>((resolve) => {
    releaseProject = resolve;
  });

  const run = runCreate(
    items(),
    async (y) => {
      events.push(`create:${y}`);
    },
    () => {},
    {
      sleep: noSleep,
      awaitReady: (item) => {
        if (item.kind !== 'Project') {
          return undefined;
        }
        events.push('gate:start');
        return projectReady.then(() => {
          events.push('gate:done');
        });
      }
    }
  );

  // A macrotask flushes every pending microtask, so the engine is parked on the
  // gate by now -- nothing past the Project can have run.
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(events).toEqual(['create:a', 'gate:start']);

  releaseProject();
  expect(await run).toBe(true);
  expect(events).toEqual(['create:a', 'gate:start', 'gate:done', 'create:b', 'create:c']);
});

test('runCreate halts when a readiness gate fails, without re-creating on retry', async () => {
  const applied: string[] = [];
  let captured: ProgressItem[] = [];

  const ok = await runCreate(
    items(),
    async (y) => {
      applied.push(y);
    },
    (p) => {
      captured = p;
    },
    {
      sleep: noSleep,
      awaitReady: (item) =>
        item.kind === 'Project' ? Promise.reject(new Error('not ready in time')) : undefined
    }
  );

  expect(ok).toBe(false);
  expect(applied).toEqual(['a']);
  expect(captured[0].state).toBe('error');
  expect(captured[0].message).toBe('not ready in time');
  // Created, so a retry must not re-apply it -- only re-run its gate.
  expect(captured[0].created).toBe(true);
  expect(captured[1].message).toBe('Halted by prior failure');

  applied.length = 0;
  const retried = await runCreate(
    mergeForRetry(captured, items()),
    async (y) => {
      applied.push(y);
    },
    () => {},
    { sleep: noSleep, awaitReady: () => undefined }
  );
  expect(retried).toBe(true);
  expect(applied).toEqual(['b', 'c']);
});

// Two Stages named "dev": the first is created, the second can only ever fail.
// The created one must keep its own status rather than inheriting the failure,
// and renaming the duplicate must not re-apply it into an "already exists" that
// no further retry can get past.
test('a duplicate name does not poison the resource it duplicates', async () => {
  const previous: ProgressItem[] = [
    { kind: 'Stage', name: 'dev', yaml: 'a', state: 'done', message: 'Created', created: true },
    {
      kind: 'Stage',
      name: 'dev',
      ordinal: 1,
      yaml: 'b',
      state: 'error',
      message: 'stages.kargo.akuity.io "dev" already exists'
    }
  ];
  // The user renames the duplicate to "staging" and hits Retry.
  const fresh = toProgressItems([
    { kind: 'Stage', name: 'dev', yaml: 'a' },
    { kind: 'Stage', name: 'staging', yaml: 'b2' }
  ]);

  const merged = mergeForRetry(previous, fresh);
  expect(merged.map((i) => [i.name, i.state])).toEqual([
    ['dev', 'done'],
    ['staging', 'pending']
  ]);

  const applied: string[] = [];
  const ok = await runCreate(
    merged,
    async (y) => {
      applied.push(y);
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(true);
  // Only the renamed Stage is applied; the created one is left alone.
  expect(applied).toEqual(['b2']);
});

test('mergeForRetry carries `created` forward for items that never became ready', () => {
  const previous: ProgressItem[] = [
    { kind: 'Project', name: 'p', yaml: 'a', state: 'error', message: 'not ready', created: true }
  ];
  const merged = mergeForRetry(
    previous,
    toProgressItems([{ kind: 'Project', name: 'p', yaml: 'a' }])
  );
  expect(merged[0].state).toBe('pending');
  expect(merged[0].created).toBe(true);
});

test('createdProjectName reads the Project name from the submitted items', () => {
  expect(createdProjectName(items())).toBe('p');
});

// Callers redirect to this name, so a missing or blank one must be reported as
// absent rather than as an empty name that would build a broken route.
test('createdProjectName is undefined when no named Project was submitted', () => {
  expect(
    createdProjectName(toProgressItems([{ kind: 'Warehouse', name: 'w', yaml: 'c' }]))
  ).toBeUndefined();
  expect(
    createdProjectName(toProgressItems([{ kind: 'Project', name: '', yaml: 'a' }]))
  ).toBeUndefined();
  expect(createdProjectName([])).toBeUndefined();
});

test('errorMessage extracts a readable message', () => {
  expect(errorMessage(apiError(422, 'Unprocessable', { error: 'nope' }))).toBe('nope');
  expect(errorMessage(apiError(500, 'Server Error', ''))).toBe('Server Error');
  expect(errorMessage(new Error('plain'))).toBe('plain');
});
