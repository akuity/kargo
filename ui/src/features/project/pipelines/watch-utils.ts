import { authTokenKey } from '@ui/config/auth';

export const getBaseUrl = () => (import.meta.env.VITE_API_URL as string | undefined) || '';

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
