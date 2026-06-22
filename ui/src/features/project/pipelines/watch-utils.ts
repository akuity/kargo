import { authTokenKey } from '@ui/config/auth';
import { getBaseUrl } from '@ui/lib/api/custom-fetch';

// Code the server emits (as `event: error`) when the resourceVersion used to
// seed a watch is older than the API server's watch window. Clients respond by
// relisting to obtain a fresh, watchable resourceVersion.
export const WATCH_ERROR_EXPIRED = 'out_of_range';

export type SSEWatchError = { code: string; message: string };

export type SSEWatchEvent<T> = { type: string; object: T } | { error: SSEWatchError };

export async function* readSSEStream<T>(
  url: string,
  signal: AbortSignal
): AsyncGenerator<SSEWatchEvent<T>> {
  const token = localStorage.getItem(authTokenKey);
  const response = await fetch(`${getBaseUrl()}${url}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    signal
  });

  if (!response.ok || !response.body) {
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true });
      const parts = buffer.split('\n\n');
      buffer = parts.pop() ?? '';

      for (const part of parts) {
        const lines = part.split('\n');
        const dataLine = lines.find((l) => l.startsWith('data: '));
        if (!dataLine) {
          continue;
        }
        const isError = lines.some((l) => l.startsWith('event: error'));
        try {
          const parsed = JSON.parse(dataLine.slice(6));
          yield isError
            ? { error: parsed as SSEWatchError }
            : (parsed as { type: string; object: T });
        } catch (_) {
          // skip malformed events
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

// runSeededWatch opens a Kubernetes-style list-then-watch SSE stream seeded from
// the list's resourceVersion, so the API server does not replay every existing
// object as an ADDED event. The seed resourceVersion is read once per connection
// from `seedResourceVersion()` (kept out of any React effect deps by the caller
// so a cache-advancing event never tears the stream down). If the seed is too
// old, the server reports `WATCH_ERROR_EXPIRED`; we then `relist()` for a fresh
// resourceVersion and reopen. A clean close or generic error stops the stream
// (matching the pre-seeding behavior — no auto-reconnect).
export async function runSeededWatch<T>(params: {
  signal: AbortSignal;
  buildUrl: (resourceVersion: string) => string;
  seedResourceVersion: () => string | undefined;
  relist: () => Promise<string | undefined>;
  onEvent: (type: string, object: T) => void;
}): Promise<void> {
  let resourceVersion = params.seedResourceVersion() || '';

  for (;;) {
    let expired = false;
    try {
      for await (const event of readSSEStream<T>(params.buildUrl(resourceVersion), params.signal)) {
        if ('error' in event) {
          if (event.error.code === WATCH_ERROR_EXPIRED) {
            expired = true;
            break;
          }
          continue;
        }
        params.onEvent(event.type, event.object);
      }
    } catch (_) {
      // Aborted unmount or a network failure: stop.
      return;
    }

    if (params.signal.aborted || !expired) {
      return;
    }

    resourceVersion = (await params.relist()) || '';
  }
}

// isNewerResourceVersion reports whether incoming is a strictly newer Kubernetes
// resourceVersion than existing. resourceVersions are officially opaque, so we
// only compare numerically (their actual encoding today) and otherwise fail open
// (treat as newer) to avoid suppressing a real update.
function isNewerResourceVersion(incoming?: string, existing?: string): boolean {
  if (!incoming) {
    return false;
  }
  if (!existing) {
    return true;
  }
  try {
    return BigInt(incoming) > BigInt(existing);
  } catch (_) {
    return true;
  }
}

// Reads an SSE stream of raw text chunks (e.g. AnalysisRun logs). The server
// writes each line of a chunk as its own `data:` line, so rejoining the data
// lines with `\n` reconstructs the original chunk verbatim. Unlike
// readSSEStream, a non-OK response throws so callers can surface the error.
export async function* readSSETextStream(url: string, signal: AbortSignal): AsyncGenerator<string> {
  const token = localStorage.getItem(authTokenKey);
  const response = await fetch(`${getBaseUrl()}${url}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    signal
  });

  if (!response.ok) {
    let message = response.statusText;
    try {
      const body = await response.json();
      message = body?.error || body?.message || message;
    } catch (_) {
      // keep status text
    }
    throw new Error(message);
  }

  if (!response.body) {
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true });
      const events = buffer.split('\n\n');
      buffer = events.pop() ?? '';

      for (const event of events) {
        yield event
          .split('\n')
          .filter((line) => line.startsWith('data: '))
          .map((line) => line.slice(6))
          .join('\n');
      }
    }
  } finally {
    reader.releaseLock();
  }
}

// Coalesces rapid invocations into a single trailing call. Watch streams emit
// bursts of events (e.g. during a refresh storm); without this the per-event
// callback would fire once per event, and the pipeline graph's recompute
// debounce would be starved by the constant stream of redraw triggers. Mirrors
// the throttling the pre-REST watch implementation had. The default zero delay
// coalesces every event parsed from a single network chunk, since the stream
// consumer processes them synchronously before the next read.
export function debounce<A extends unknown[]>(
  fn: (...args: A) => void,
  delayMs = 0
): { call: (...args: A) => void; cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | undefined;
  return {
    call: (...args: A) => {
      clearTimeout(timer);
      timer = setTimeout(() => fn(...args), delayMs);
    },
    cancel: () => clearTimeout(timer)
  };
}

export function upsertOrDelete<
  T extends { metadata?: { name?: string; resourceVersion?: string } }
>(items: T[], item: T, eventType: string): T[] {
  const index = items.findIndex((i) => i.metadata?.name === item.metadata?.name);
  if (eventType === 'DELETED') {
    return index !== -1 ? [...items.slice(0, index), ...items.slice(index + 1)] : items;
  }
  if (index === -1) {
    return [...items, item];
  }
  // Skip a replayed ADDED for an object we already hold at the same or newer
  // resourceVersion. Seeding should prevent these server-side; this is a
  // belt-and-suspenders guard for an unseeded or just-relisted watch. MODIFIED
  // events always carry a newer resourceVersion and are applied unconditionally.
  if (
    eventType === 'ADDED' &&
    !isNewerResourceVersion(item.metadata?.resourceVersion, items[index].metadata?.resourceVersion)
  ) {
    return items;
  }
  return [...items.slice(0, index), item, ...items.slice(index + 1)];
}
