import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';

import { authTokenKey, refreshTokenKey } from '../../config/auth';

import { customFetch } from './custom-fetch';

// The tests run in Vitest's default node environment, which has neither
// localStorage nor window.
const fakeLocalStorage = () => {
  const store = new Map<string, string>();
  return {
    getItem: (key: string) => (store.has(key) ? (store.get(key) as string) : null),
    setItem: (key: string, value: string) => void store.set(key, String(value)),
    removeItem: (key: string) => void store.delete(key),
    clear: () => store.clear(),
    key: (index: number) => [...store.keys()][index] ?? null,
    get length() {
      return store.size;
    }
  };
};

// Builds an unsigned JWT whose exp is `offsetSeconds` from now. customFetch
// only parses the payload, so a real signature is unnecessary.
const jwt = (offsetSeconds: number) => {
  const payload = { exp: Math.floor(Date.now() / 1000) + offsetSeconds };
  return `header.${btoa(JSON.stringify(payload))}.signature`;
};

describe('customFetch', () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  let replaceMock: ReturnType<typeof vi.fn>;

  const stubWindow = (pathname: string) => {
    replaceMock = vi.fn();
    vi.stubGlobal('window', {
      __KARGO_BASE_PATH__: '',
      location: { pathname, replace: replaceMock }
    });
  };

  beforeEach(() => {
    vi.stubGlobal('localStorage', fakeLocalStorage());
    fetchMock = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    stubWindow('/');
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  test('navigates to the renewal page when the token has expired and a refresh token exists', async () => {
    localStorage.setItem(authTokenKey, jwt(-60));
    localStorage.setItem(refreshTokenKey, 'refresh-1');

    await expect(customFetch('/v1beta1/projects')).rejects.toMatchObject({ status: 401 });
    expect(replaceMock).toHaveBeenCalledWith('/token-renew?redirectTo=/');
    expect(fetchMock).not.toHaveBeenCalled();
  });

  test('does not block requests to exempt endpoints on an expired token', async () => {
    localStorage.setItem(authTokenKey, jwt(-60));
    localStorage.setItem(refreshTokenKey, 'refresh-1');

    await customFetch('/v1beta1/system/public-server-config');

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][1].headers).not.toHaveProperty('Authorization');
    expect(replaceMock).not.toHaveBeenCalled();
  });

  test('ignores the query string when matching exempt endpoints', async () => {
    localStorage.setItem(authTokenKey, jwt(-60));
    localStorage.setItem(refreshTokenKey, 'refresh-1');

    await customFetch('/v1beta1/system/public-server-config?ts=1');

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(replaceMock).not.toHaveBeenCalled();
  });

  test('does not end the session on a 401 from an exempt endpoint', async () => {
    localStorage.setItem(authTokenKey, jwt(-60));
    localStorage.setItem(refreshTokenKey, 'refresh-1');
    fetchMock.mockResolvedValue(
      new Response('{"error":"invalid token"}', {
        status: 401,
        headers: { 'content-type': 'application/json' }
      })
    );

    await expect(customFetch('/v1beta1/login')).rejects.toMatchObject({ status: 401 });
    expect(localStorage.getItem(authTokenKey)).not.toBeNull();
    expect(localStorage.getItem(refreshTokenKey)).toBe('refresh-1');
    expect(replaceMock).not.toHaveBeenCalled();
  });

  test('ends the session on a 401 received elsewhere', async () => {
    localStorage.setItem(authTokenKey, jwt(3600));
    localStorage.setItem(refreshTokenKey, 'refresh-1');
    fetchMock.mockResolvedValue(
      new Response('{"error":"invalid token"}', {
        status: 401,
        headers: { 'content-type': 'application/json' }
      })
    );

    await expect(customFetch('/v1beta1/projects')).rejects.toMatchObject({ status: 401 });
    expect(localStorage.getItem(authTokenKey)).toBeNull();
    expect(replaceMock).toHaveBeenCalledWith('/login?redirectTo=/');
  });
});
