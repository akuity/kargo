/**
 * Custom fetch wrapper for the Kargo REST API.
 *
 * This mutator is used by orval-generated hooks to:
 * - Add the base URL from environment/config
 * - Include authentication headers
 * - Handle common error scenarios
 *
 * Orval generates hooks that expect a response envelope:
 * { data: T, status: number, headers: Headers }
 */

import { authTokenKey, redirectToQueryParam, refreshTokenKey } from '@ui/config/auth';
import { paths } from '@ui/config/paths';

const getBaseUrl = (): string => {
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  return '';
};

const logout = () => {
  localStorage.removeItem(authTokenKey);
  localStorage.removeItem(refreshTokenKey);
  window.location.replace(`${paths.login}?${redirectToQueryParam}=${window.location.pathname}`);
};

const renewToken = () => {
  window.location.replace(
    `${paths.tokenRenew}?${redirectToQueryParam}=${window.location.pathname}`
  );
};

/**
 * Custom fetch function used by all generated API hooks.
 *
 * Returns a response envelope { data, status, headers } as expected
 * by orval-generated hooks.
 *
 * @param url - The API endpoint path (e.g., "/v1beta1/projects")
 * @param options - The fetch options (method, body, headers, etc.)
 * @returns Promise resolving to the response envelope
 */
export const customFetch = async <T>(url: string, options?: RequestInit): Promise<T> => {
  const baseUrl = getBaseUrl();
  const fullUrl = `${baseUrl}${url}`;

  const token = localStorage.getItem(authTokenKey);
  const refreshToken = localStorage.getItem(refreshTokenKey);

  if (token) {
    let isTokenExpired: boolean;
    try {
      isTokenExpired = Date.now() >= JSON.parse(atob(token.split('.')[1])).exp * 1000;
    } catch (_) {
      logout();
      throw new ApiError(401, 'Unauthorized', 'Invalid token');
    }

    if (isTokenExpired && refreshToken) {
      renewToken();
      throw new ApiError(401, 'Unauthorized', 'Token expired');
    }

    if (isTokenExpired && !refreshToken) {
      logout();
      throw new ApiError(401, 'Unauthorized', 'Token expired');
    }
  }

  const headers: Record<string, string> = {
    'Content-Type': 'application/json'
  };

  // Preserve any explicitly passed headers (may override Content-Type for text/plain bodies)
  if (options?.headers) {
    const incoming = options.headers;
    if (incoming instanceof Headers) {
      incoming.forEach((value, key) => {
        headers[key] = value;
      });
    } else if (Array.isArray(incoming)) {
      for (const [key, value] of incoming) {
        headers[key] = value;
      }
    } else {
      Object.assign(headers, incoming);
    }
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(fullUrl, {
    ...options,
    headers
  });

  if (response.status === 401) {
    logout();
  }

  if (!response.ok) {
    let errorBody: unknown;
    try {
      errorBody = await response.json();
    } catch {
      errorBody = await response.text();
    }
    throw new ApiError(response.status, response.statusText, errorBody);
  }

  if (response.status === 204) {
    return { data: undefined, status: 204, headers: response.headers } as T;
  }

  const contentType = response.headers.get('content-type') ?? '';
  let data: unknown;
  if (contentType.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  return { data, status: response.status, headers: response.headers } as T;
};

/**
 * Custom error class for API errors.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly statusText: string,
    public readonly body: unknown
  ) {
    super(`API Error: ${status} ${statusText}`);
    this.name = 'ApiError';
  }

  is(status: number): boolean {
    return this.status === status;
  }

  isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  isServerError(): boolean {
    return this.status >= 500;
  }

  isUnauthorized(): boolean {
    return this.status === 401;
  }

  isForbidden(): boolean {
    return this.status === 403;
  }

  isNotFound(): boolean {
    return this.status === 404;
  }
}

export default customFetch;
