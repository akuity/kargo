import { describe, expect, test } from 'vitest';

import {
  claimExpired,
  claimRows,
  claimText,
  claimValue,
  isBuiltInAdmin,
  relativeTime,
  signInMethod
} from './claims';

describe('claimText', () => {
  test('keeps a string as it is', () => {
    expect(claimText('admin')).toBe('admin');
  });

  test('writes anything else as JSON', () => {
    expect(claimText(1795875979)).toBe('1795875979');
    expect(claimText(true)).toBe('true');
    expect(claimText(['a', 'b'])).toBe('["a","b"]');
    expect(claimText({ a: 1 })).toBe('{"a":1}');
    expect(claimText(null)).toBe('null');
  });

  test('copes with a value JSON cannot hold', () => {
    expect(claimText(undefined)).toBe('');
  });
});

describe('claimValue', () => {
  test('reads a time claim as a date', () => {
    expect(claimValue('exp', 1795875979)).toEqual({
      kind: 'time',
      date: new Date(1795875979 * 1000),
      seconds: 1795875979
    });
  });

  test('reads a time claim that is not a number as text', () => {
    expect(claimValue('exp', 'soon')).toEqual({ kind: 'text', text: 'soon' });
  });

  test('reads a time too far off to be a date as text', () => {
    expect(claimValue('exp', 1e20)).toEqual({ kind: 'text', text: '100000000000000000000' });
  });

  test('leaves a number in any other claim alone', () => {
    expect(claimValue('custom', 1795875979)).toEqual({ kind: 'text', text: '1795875979' });
  });

  test('reads an array of plain values as a list', () => {
    expect(claimValue('groups', ['devs', 1, false])).toEqual({
      kind: 'list',
      items: ['devs', '1', 'false']
    });
  });

  test('reads an array holding objects as JSON', () => {
    expect(claimValue('roles', [{ a: 1 }])).toEqual({ kind: 'text', text: '[{"a":1}]' });
  });
});

describe('claimRows', () => {
  test('lists the known claims in their order, then the rest by name', () => {
    const rows = claimRows({
      zeta: 1,
      jti: 'id',
      exp: 2,
      alpha: 'a',
      sub: 'admin',
      iat: 1,
      iss: 'https://kargo.example.com'
    });
    expect(rows.map((r) => r.key)).toEqual(['iss', 'sub', 'iat', 'exp', 'jti', 'alpha', 'zeta']);
  });

  test('labels and explains a known claim', () => {
    const [row] = claimRows({ iss: 'https://kargo.example.com' });
    expect(row.label).toBe('Issuer');
    expect(row.description).toBe('Who issued this token');
  });

  test('labels an unknown claim by its name, with nothing to explain it', () => {
    const [row] = claimRows({ tenant: 'acme' });
    expect(row.label).toBe('tenant');
    expect(row.description).toBeUndefined();
  });

  test('does not take a claim named after an Object member for a known one', () => {
    const [row] = claimRows({ constructor: 'x' });
    expect(row.label).toBe('constructor');
    expect(row.description).toBeUndefined();
  });

  test('copies a claim as the token has it', () => {
    const rows = claimRows({ exp: 1795875979, groups: ['devs'] });
    expect(rows.map((r) => r.copy)).toEqual(['["devs"]', '1795875979']);
  });
});

describe('relativeTime', () => {
  const now = new Date('2026-11-27T15:00:00Z');

  test('says how long until a time to come', () => {
    expect(relativeTime(new Date('2026-11-28T15:00:00Z'), now)).toBe('in 1 day');
  });

  test('says how long since a time gone by', () => {
    expect(relativeTime(new Date('2026-11-27T13:00:00Z'), now)).toBe('about 2 hours ago');
  });
});

describe('claimExpired', () => {
  const now = new Date('2026-11-27T15:00:00Z');
  const at = (iso: string) => new Date(iso).getTime() / 1000;

  test('says an expiry gone by has expired', () => {
    const [claim] = claimRows({ exp: at('2026-11-27T14:59:00Z') });
    expect(claimExpired(claim, now)).toBe(true);
  });

  test('says an expiry at this very moment has expired', () => {
    const [claim] = claimRows({ exp: at('2026-11-27T15:00:00Z') });
    expect(claimExpired(claim, now)).toBe(true);
  });

  test('says an expiry to come has not', () => {
    const [claim] = claimRows({ exp: at('2026-11-27T15:01:00Z') });
    expect(claimExpired(claim, now)).toBe(false);
  });

  test('never says another time claim has expired', () => {
    const [claim] = claimRows({ iat: at('2026-11-27T14:00:00Z') });
    expect(claimExpired(claim, now)).toBe(false);
  });

  test('never says an expiry that is not a time has expired', () => {
    const [claim] = claimRows({ exp: 'soon' });
    expect(claimExpired(claim, now)).toBe(false);
  });
});

describe('isBuiltInAdmin', () => {
  const admin = { sub: 'admin', iss: 'https://kargo.example.com' };
  const sso = { sub: 'admin', iss: 'https://kargo.example.com/dex/' };
  const config = {
    adminAccountEnabled: true,
    oidcConfig: { issuerUrl: 'https://kargo.example.com/dex' }
  };

  test('is never a subject other than admin', () => {
    expect(isBuiltInAdmin({ ...admin, sub: 'jane' }, config)).toBe(false);
    expect(isBuiltInAdmin(null, config)).toBe(false);
  });

  test('goes by the subject alone until the config is in', () => {
    expect(isBuiltInAdmin(sso)).toBe(true);
  });

  test('is never anyone when there is no admin account', () => {
    expect(isBuiltInAdmin(admin, { ...config, adminAccountEnabled: false })).toBe(false);
  });

  test('is not a user single sign-on calls admin', () => {
    expect(isBuiltInAdmin(sso, config)).toBe(false);
  });

  test('is admin with any other issuer', () => {
    expect(isBuiltInAdmin(admin, config)).toBe(true);
  });

  test('is admin when single sign-on is off', () => {
    expect(isBuiltInAdmin(admin, { adminAccountEnabled: true })).toBe(true);
  });
});

describe('signInMethod', () => {
  test('names the built-in admin login', () => {
    expect(signInMethod({ sub: 'admin', iss: 'https://kargo.example.com' }, true)).toBe(
      'built-in admin login'
    );
  });

  test('names the identity provider by its host', () => {
    expect(signInMethod({ sub: 'jane', iss: 'https://dex.example.com/dex' }, false)).toBe(
      'single sign-on (dex.example.com)'
    );
  });

  test('names an issuer that is not a URL as it is', () => {
    expect(signInMethod({ sub: 'jane', iss: 'my-idp' }, false)).toBe('single sign-on (my-idp)');
  });

  test('names an issuer that parses as a URL with no host as it is', () => {
    expect(signInMethod({ sub: 'jane', iss: 'localhost:5556' }, false)).toBe(
      'single sign-on (localhost:5556)'
    );
    expect(signInMethod({ sub: 'jane', iss: 'urn:example:idp' }, false)).toBe(
      'single sign-on (urn:example:idp)'
    );
  });

  test('copes with no issuer', () => {
    expect(signInMethod({ sub: 'jane', iss: '' }, false)).toBe('single sign-on');
    expect(signInMethod(null, false)).toBe('single sign-on');
  });
});
