/**
 * Custom fetch wrapper for the Kargo REST API.
 *
 * This mutator is used by orval-generated hooks to:
 * - Add the base URL from environment/config
 * - Include authentication headers
 * - Handle common error scenarios
 *
 * The UI team should customize this based on how authentication
 * is handled in the current application.
 */

// TODO: Import your auth token getter from the appropriate location
// import { getAuthToken } from '@/auth';

/**
 * Get the API base URL from environment or default to current origin.
 * Adjust this based on your deployment configuration.
 */
const getBaseUrl = (): string => {
  // In development, you might use a different API URL
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  // In production, API is typically on the same origin
  return '';
};

/**
 * Custom fetch function used by all generated API hooks.
 *
 * Orval calls this function with (url, options) signature.
 *
 * @param url - The API endpoint path (e.g., "/v2/projects")
 * @param options - The fetch options (method, body, headers, etc.)
 * @returns Promise resolving to the parsed response data
 */
export const customFetch = async <T>(url: string, options?: RequestInit): Promise<T> => {
  const baseUrl = getBaseUrl();
  const fullUrl = `${baseUrl}${url}`;

  // TODO: Get the auth token from your auth state/context
  // const token = getAuthToken();
  const token: string | null = null; // Placeholder - implement auth token retrieval

  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options?.headers
  };

  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(fullUrl, {
    ...options,
    headers
  });

  // Handle non-2xx responses
  if (!response.ok) {
    // Try to parse error body for more details
    let errorBody: unknown;
    try {
      errorBody = await response.json();
    } catch {
      errorBody = await response.text();
    }

    throw new ApiError(response.status, response.statusText, errorBody);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  // Parse and return JSON response
  return response.json();
};

/**
 * Custom error class for API errors.
 * Provides structured error information for error handling in the UI.
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

  /**
   * Check if this is a specific HTTP status code.
   */
  is(status: number): boolean {
    return this.status === status;
  }

  /**
   * Check if this is a client error (4xx).
   */
  isClientError(): boolean {
    return this.status >= 400 && this.status < 500;
  }

  /**
   * Check if this is a server error (5xx).
   */
  isServerError(): boolean {
    return this.status >= 500;
  }

  /**
   * Check if this is an authentication error (401).
   */
  isUnauthorized(): boolean {
    return this.status === 401;
  }

  /**
   * Check if this is a forbidden error (403).
   */
  isForbidden(): boolean {
    return this.status === 403;
  }

  /**
   * Check if this is a not found error (404).
   */
  isNotFound(): boolean {
    return this.status === 404;
  }
}

export default customFetch;
