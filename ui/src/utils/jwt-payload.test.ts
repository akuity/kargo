import { describe, expect, test } from 'vitest';

import { parseJwtPayload } from './jwt-payload';

/** JWT-style Base64URL of UTF-8 JSON (no `=` padding, `-`/`_` alphabet). */
function base64UrlEncodeUtf8Json(value: unknown): string {
  const json = JSON.stringify(value);
  const bytes = new TextEncoder().encode(json);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

function makeJwt(payload: Record<string, unknown>): string {
  const header = base64UrlEncodeUtf8Json({ alg: 'none', typ: 'JWT' });
  const payloadSegment = base64UrlEncodeUtf8Json(payload);
  return `${header}.${payloadSegment}.signature`;
}

test('parseJwtPayload decodes UTF-8 claims (diacritics)', () => {
  const exp = 2_000_000_000;
  const token = makeJwt({ sub: 'user-1', name: 'José García', exp });

  expect(parseJwtPayload(token)).toEqual({
    sub: 'user-1',
    name: 'José García',
    exp
  });
});

/**
 * Regression: naive `JSON.parse(atob(segment))` mishandles UTF-8 bytes for some
 * six-character family names ending in `o` + a diacritic (e.g. ï, ñ), while
 * other lengths/compositions decode fine.
 */
describe('parseJwtPayload familyName regression cases', () => {
  test.each(['abcdef', 'abcdeï', 'abcdeoï', 'abcdoï', 'abcdoñ'] as const)(
    'round-trips familyName %s',
    (familyName) => {
      const payload = { sub: 'u1', familyName, exp: 2_000_000_000 };
      const token = makeJwt(payload);
      expect(parseJwtPayload<typeof payload>(token)).toEqual(payload);
    }
  );
});

test('parseJwtPayload handles Base64URL alphabet (+ → -) and restored padding', () => {
  // Single-character claim whose UTF-8 JSON base64 includes "+" (must become "-" in JWT).
  const payload = { n: String.fromCodePoint(0x83e) };
  const token = makeJwt(payload);
  const segment = token.split('.')[1];
  expect(segment).toContain('-');

  expect(parseJwtPayload(token)).toEqual(payload);
});

test('parseJwtPayload throws on missing payload segment', () => {
  expect(() => parseJwtPayload('only-one')).toThrow(/missing payload/);
});

test('parseJwtPayload throws on invalid base64', () => {
  expect(() => parseJwtPayload('a.b!!!.c')).toThrow();
});
