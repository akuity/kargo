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

test('apiErrorBodyMessage unwraps a JSON body', () => {
  expect(apiErrorBodyMessage({ message: 'bad name' })).toBe('bad name');
  expect(apiErrorBodyMessage({ error: 'already exists' })).toBe('already exists');
  expect(apiErrorBodyMessage({ message: 'm', error: 'e' })).toBe('m');
});

// customFetch falls back to response.text() when the error body isn't JSON, so
// the raw text is often the only thing the server told us.
test('apiErrorBodyMessage passes a plain-text body through', () => {
  expect(apiErrorBodyMessage('namespace "p" not found')).toBe('namespace "p" not found');
});

test('apiErrorBodyMessage returns undefined when the body says nothing usable', () => {
  expect(apiErrorBodyMessage('')).toBeUndefined();
  expect(apiErrorBodyMessage({})).toBeUndefined();
  expect(apiErrorBodyMessage(null)).toBeUndefined();
  expect(apiErrorBodyMessage(undefined)).toBeUndefined();
  expect(apiErrorBodyMessage([{ message: 'x' }])).toBeUndefined();
  // Non-string message fields aren't messages -- coercing them yields noise
  // like "123" or "[object Object]" in place of a real error.
  expect(apiErrorBodyMessage({ message: 123 })).toBeUndefined();
  expect(apiErrorBodyMessage({ message: '' })).toBeUndefined();
  expect(apiErrorBodyMessage({ error: { message: 'deep' } })).toBeUndefined();
});
