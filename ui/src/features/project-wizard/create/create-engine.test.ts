import { expect, test } from 'vitest';

import {
  ProgressItem,
  errorMessage,
  isNamespaceNotReady,
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

test('runCreate retries a retryable (namespace-not-ready) error then succeeds', async () => {
  let attempts = 0;
  const ok = await runCreate(
    toProgressItems([{ kind: 'Secret', name: 's', yaml: 'b' }]),
    async () => {
      attempts++;
      if (attempts < 3) {
        throw apiError(404, 'Not Found', 'namespace "p" not found');
      }
    },
    () => {},
    { sleep: noSleep }
  );
  expect(ok).toBe(true);
  expect(attempts).toBe(3);
});

test('runCreate does not retry non-retryable errors', async () => {
  let attempts = 0;
  const ok = await runCreate(
    toProgressItems([{ kind: 'Secret', name: 's', yaml: 'b' }]),
    async () => {
      attempts++;
      throw apiError(422, 'Unprocessable', 'invalid');
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

test('isNamespaceNotReady only matches namespace-not-found ApiErrors', () => {
  expect(isNamespaceNotReady(apiError(404, 'x', 'namespace "p" not found'))).toBe(true);
  expect(isNamespaceNotReady(apiError(422, 'x', 'invalid field'))).toBe(false);
  expect(isNamespaceNotReady(new Error('namespace not found'))).toBe(false);
});

test('errorMessage extracts a readable message', () => {
  expect(errorMessage(apiError(422, 'Unprocessable', { message: 'nope' }))).toBe('nope');
  expect(errorMessage(apiError(500, 'Server Error', ''))).toBe('Server Error');
  expect(errorMessage(new Error('plain'))).toBe('plain');
});
