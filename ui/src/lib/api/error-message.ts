import type { ResourceErrorResponse } from '@ui/gen/api/v2/models';

import type { ApiError } from './custom-fetch';

// Structural shape of the REST client's ApiError. Declared from a type-only
// import so callers can read API errors without pulling the HTTP client (and
// its localStorage/window dependencies) into their runtime graph.
type ApiErrorLike = Partial<Pick<ApiError, 'status' | 'statusText' | 'body' | 'message'>>;

export const isApiErrorLike = (error: unknown): error is ApiErrorLike =>
  typeof error === 'object' && error !== null && typeof (error as ApiErrorLike).status === 'number';

// Past this, response text is a document rather than a message -- an error page
// served by something sitting between the client and the API. Callers render
// what they get in full, so an unbounded body would fill the screen with markup.
const maxTextBodyLength = 200;

// The message the server put in an error response body, or undefined when it
// carried none worth showing. customFetch decodes the body as JSON when it can
// and falls back to raw text, so a body is either a ResourceErrorResponse --
// the one error shape the API emits, for every status -- or the response text
// of something that isn't the API, such as Gin's own "404 page not found".
//
// Callers supply their own fallback for the undefined case, because what reads
// best depends on where the message lands: ApiError.message ("API Error: 404
// Not Found") keeps the status code visible in a toast that appears on its own,
// while the bare statusText ("Not Found") reads better in a list whose rows are
// already labelled with the resource they belong to.
export const apiErrorBodyMessage = (body: unknown): string | undefined => {
  if (typeof body === 'string') {
    const text = body.trim();
    return text && text.length <= maxTextBodyLength && !text.startsWith('<') ? text : undefined;
  }
  if (body && typeof body === 'object') {
    // The generated shape is the contract; the check is because the body is
    // whatever crossed the wire, not necessarily what the API promised.
    const { error } = body as ResourceErrorResponse;
    return typeof error === 'string' && error ? error : undefined;
  }
  return undefined;
};
