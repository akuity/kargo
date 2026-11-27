import { describe, expect, test } from 'vitest';

import { asText, getDisplayName, JWTInfo } from './jwt-utils';

const jwt = (claims: Partial<JWTInfo>): JWTInfo => ({ sub: '', iss: '', ...claims });

describe('asText', () => {
  test('trims a string', () => {
    expect(asText('  jane  ')).toBe('jane');
  });

  test('drops anything that is not a string', () => {
    expect(asText(42)).toBe('');
    expect(asText({ name: 'jane' })).toBe('');
    expect(asText(undefined)).toBe('');
  });
});

describe('getDisplayName', () => {
  test('prefers the name', () => {
    expect(getDisplayName(jwt({ name: 'Jane Doe', email: 'jane@example.com', sub: 'jane' }))).toBe(
      'Jane Doe'
    );
  });

  test('falls back through username, email and subject', () => {
    expect(getDisplayName(jwt({ preferred_username: 'jdoe', sub: 'jane' }))).toBe('jdoe');
    expect(getDisplayName(jwt({ email: 'jane@example.com', sub: 'jane' }))).toBe(
      'jane@example.com'
    );
    expect(getDisplayName(jwt({ sub: 'admin' }))).toBe('admin');
  });

  test('skips blank claims', () => {
    expect(getDisplayName(jwt({ name: '  ', sub: 'jane' }))).toBe('jane');
  });

  test('says Account when nothing names the user', () => {
    expect(getDisplayName(null)).toBe('Account');
  });
});
