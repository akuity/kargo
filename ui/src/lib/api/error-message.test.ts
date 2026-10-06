import { expect, test } from 'vitest';

import { ApiError } from './custom-fetch';
import { apiErrorBodyMessage, isApiErrorLike } from './error-message';

test('isApiErrorLike accepts anything carrying a numeric status', () => {
  expect(isApiErrorLike(new ApiError(404, 'Not Found', ''))).toBe(true);
  expect(isApiErrorLike({ status: 500 })).toBe(true);
  expect(isApiErrorLike(new Error('plain'))).toBe(false);
  expect(isApiErrorLike({ status: '500' })).toBe(false);
  expect(isApiErrorLike(null)).toBe(false);
});

// Every error the API emits is a ResourceErrorResponse, whatever the status --
// see the handleError middleware in pkg/server/rest_router.go.
test('apiErrorBodyMessage unwraps a JSON body', () => {
  expect(apiErrorBodyMessage({ error: 'already exists' })).toBe('already exists');
  expect(apiErrorBodyMessage({ error: 'request body too large' })).toBe('request body too large');
});

// customFetch falls back to response.text() when the error body isn't JSON, so
// the raw text is often the only thing the server told us.
test('apiErrorBodyMessage passes a plain-text body through', () => {
  expect(apiErrorBodyMessage('namespace "p" not found')).toBe('namespace "p" not found');
  expect(apiErrorBodyMessage('  404 page not found\n')).toBe('404 page not found');
});

// A body this shape came from a proxy or ingress between the client and the
// API, not from the API. Callers render the message in full, so passing it
// through would bury the screen under an error page.
test('apiErrorBodyMessage rejects a text body that is really a document', () => {
  expect(apiErrorBodyMessage('<html><body><h1>502 Bad Gateway</h1></body></html>')).toBeUndefined();
  expect(apiErrorBodyMessage(`upstream connect error ${'x'.repeat(200)}`)).toBeUndefined();
  // Right up to the cap still reads as a message.
  expect(apiErrorBodyMessage('e'.repeat(200))).toBe('e'.repeat(200));
});

test('apiErrorBodyMessage returns undefined when the body says nothing usable', () => {
  expect(apiErrorBodyMessage('')).toBeUndefined();
  expect(apiErrorBodyMessage('   \n ')).toBeUndefined();
  expect(apiErrorBodyMessage({})).toBeUndefined();
  expect(apiErrorBodyMessage(null)).toBeUndefined();
  expect(apiErrorBodyMessage(undefined)).toBeUndefined();
  expect(apiErrorBodyMessage([{ error: 'x' }])).toBeUndefined();
  expect(apiErrorBodyMessage({ error: '' })).toBeUndefined();
  // Non-string error fields aren't messages -- coercing them yields noise like
  // "123" or "[object Object]" in place of a real error.
  expect(apiErrorBodyMessage({ error: 123 })).toBeUndefined();
  expect(apiErrorBodyMessage({ error: { message: 'deep' } })).toBeUndefined();
  // The API never sends this; a body carrying only it says nothing usable.
  expect(apiErrorBodyMessage({ message: 'not the API shape' })).toBeUndefined();
});
