import { describe, expect, test } from 'vitest';

import { ApiError } from '../../../../lib/api/custom-fetch';

import {
  createdPromotionMaxRetries,
  createdPromotionRetryDelay,
  createdPromotionState,
  isCreatedPromotion,
  shouldRetryCreatedPromotion
} from './created-promotion';

const notFound = new ApiError(404, 'Not Found', { error: 'not found' });

describe('createdPromotionState', () => {
  test('marks the promotion it was built for as just created', () => {
    expect(isCreatedPromotion(createdPromotionState('a'), 'a')).toBe(true);
    expect(isCreatedPromotion(createdPromotionState('a'), 'b')).toBe(false);
  });

  test('marks nothing when the promotion name is missing', () => {
    expect(isCreatedPromotion(createdPromotionState(undefined), '')).toBe(false);
  });
});

describe('isCreatedPromotion', () => {
  test('is true when the state names the same promotion', () => {
    expect(isCreatedPromotion({ createdPromotion: 'a' }, 'a')).toBe(true);
  });

  test('is false when the state names a different promotion', () => {
    expect(isCreatedPromotion({ createdPromotion: 'a' }, 'b')).toBe(false);
  });

  test('is false without navigation state', () => {
    expect(isCreatedPromotion(null, 'a')).toBe(false);
    expect(isCreatedPromotion(undefined, 'a')).toBe(false);
    expect(isCreatedPromotion({}, 'a')).toBe(false);
  });

  test('is false for an empty promotion name', () => {
    expect(isCreatedPromotion({ createdPromotion: '' }, '')).toBe(false);
  });
});

describe('shouldRetryCreatedPromotion', () => {
  test('retries a not found error until the retry budget is spent', () => {
    for (let failureCount = 0; failureCount < createdPromotionMaxRetries; failureCount++) {
      expect(shouldRetryCreatedPromotion(failureCount, notFound)).toBe(true);
    }
    expect(shouldRetryCreatedPromotion(createdPromotionMaxRetries, notFound)).toBe(false);
  });

  test('does not retry other API errors', () => {
    expect(shouldRetryCreatedPromotion(0, new ApiError(403, 'Forbidden', {}))).toBe(false);
    expect(shouldRetryCreatedPromotion(0, new ApiError(500, 'Internal Server Error', {}))).toBe(
      false
    );
  });

  test('does not retry non API errors', () => {
    expect(shouldRetryCreatedPromotion(0, new Error('network down'))).toBe(false);
  });
});

describe('createdPromotionRetryDelay', () => {
  test('backs off exponentially up to a cap', () => {
    expect([0, 1, 2, 3, 4, 5].map(createdPromotionRetryDelay)).toEqual([
      250, 500, 1000, 2000, 2000, 2000
    ]);
  });
});
