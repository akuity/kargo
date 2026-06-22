import { authTokenKey } from '@ui/config/auth';
import { getBaseUrl } from '@ui/lib/api/custom-fetch';

export type SSEWatchEvent<T> = { type: string; object: T };

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
        const dataLine = part.split('\n').find((l) => l.startsWith('data: '));
        if (!dataLine) {
          continue;
        }
        try {
          yield JSON.parse(dataLine.slice(6)) as SSEWatchEvent<T>;
        } catch (_) {
          // skip malformed events
        }
      }
    }
  } finally {
    reader.releaseLock();
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

export function upsertOrDelete<T extends { metadata?: { name?: string } }>(
  items: T[],
  item: T,
  eventType: string
): T[] {
  const index = items.findIndex((i) => i.metadata?.name === item.metadata?.name);
  if (eventType === 'DELETED') {
    return index !== -1 ? [...items.slice(0, index), ...items.slice(index + 1)] : items;
  }
  return index !== -1
    ? [...items.slice(0, index), item, ...items.slice(index + 1)]
    : [...items, item];
}
