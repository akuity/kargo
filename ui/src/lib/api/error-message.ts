import type { ApiError } from './custom-fetch';

// Structural shape of the REST client's ApiError. Declared from a type-only
// import so callers can read API errors without pulling the HTTP client (and
// its localStorage/window dependencies) into their runtime graph.
type ApiErrorLike = Partial<Pick<ApiError, 'status' | 'statusText' | 'body' | 'message'>>;

export const isApiErrorLike = (error: unknown): error is ApiErrorLike =>
  typeof error === 'object' && error !== null && typeof (error as ApiErrorLike).status === 'number';

// The message the server put in an error response body, or undefined when it
// carried none worth showing. customFetch decodes the body as JSON when it can
// and falls back to raw text, so a body is either a decoded value or the
// response text -- both shapes are unwrapped here.
//
// Callers supply their own fallback for the undefined case, because what reads
// best depends on where the message lands: ApiError.message ("API Error: 404
// Not Found") keeps the status code visible in a toast that appears on its own,
// while the bare statusText ("Not Found") reads better in a list whose rows are
// already labelled with the resource they belong to.
export const apiErrorBodyMessage = (body: unknown): string | undefined => {
  if (typeof body === 'string' && body) {
    return body;
  }
  if (body && typeof body === 'object') {
    const message =
      (body as { message?: unknown; error?: unknown }).message ??
      (body as { error?: unknown }).error;
    if (typeof message === 'string' && message) {
      return message;
    }
  }
  return undefined;
};
