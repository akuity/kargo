import { describe, expect, it } from 'vitest';

import { serializeParams } from './params-serializer';

describe('serializeParams', () => {
  it('omits an empty array entirely', () => {
    // Regression: a comma-joined serializer emits `freightOrigins=`, which the
    // server reads as a filter on an empty origin name and matches nothing.
    expect(serializeParams({ freightOrigins: [] })).toBe('');
  });

  it('repeats the key for each array item rather than joining with commas', () => {
    expect(serializeParams({ freightOrigins: ['wh1', 'wh2'] })).toBe(
      'freightOrigins=wh1&freightOrigins=wh2'
    );
  });

  it('serializes a single-item array as one parameter', () => {
    expect(serializeParams({ freightOrigins: ['wh1'] })).toBe('freightOrigins=wh1');
  });

  it('serializes scalar values', () => {
    expect(serializeParams({ project: 'kargo', limit: 10, watch: true })).toBe(
      'project=kargo&limit=10&watch=true'
    );
  });

  it('skips undefined values but preserves null as the string "null"', () => {
    expect(serializeParams({ a: undefined, b: null })).toBe('b=null');
  });

  it('returns an empty string for undefined and empty params', () => {
    expect(serializeParams(undefined)).toBe('');
    expect(serializeParams({})).toBe('');
  });

  it('percent-encodes keys and values', () => {
    expect(serializeParams({ 'a b': 'c&d' })).toBe('a+b=c%26d');
  });
});
