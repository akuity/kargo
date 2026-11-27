import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, test } from 'vitest';

import { TokenClaimsCard } from './token-claims-card';

// 2000-01-01 and 2100-01-01: well behind and well ahead of any clock that
// runs these tests.
const PAST = 946684800;
const FUTURE = 4102444800;

const html = (claims: Record<string, unknown>) =>
  renderToStaticMarkup(<TokenClaimsCard claims={claims} />);

// The card as text, the way it reads: tags and React's text separators gone.
const text = (claims: Record<string, unknown>) => html(claims).replace(/<[^>]*>/g, '');

describe('TokenClaimsCard', () => {
  test('lists the claims readably, known ones first', () => {
    const out = text({ tenant: 'acme', sub: 'jane', iss: 'https://dex.example.com' });
    expect(out.indexOf('Issuer')).toBeLessThan(out.indexOf('Subject'));
    expect(out.indexOf('Subject')).toBeLessThan(out.indexOf('tenant'));
    expect(out).toContain('Who issued this token');
    expect(out).toContain('https://dex.example.com');
  });

  test('says when a token has expired', () => {
    expect(text({ exp: PAST })).toContain('expired');
  });

  test('says nothing of expiry for a token still good', () => {
    const out = text({ exp: FUTURE });
    expect(out).not.toContain('expired');
    expect(out).toContain(`Unix ${FUTURE}`);
  });

  test('shows each group', () => {
    const out = text({ groups: ['devs', 'ops'] });
    expect(out).toContain('devs');
    expect(out).toContain('ops');
  });

  test('escapes markup in a claim rather than rendering it', () => {
    const out = html({ name: '<img src=x onerror=alert(1)>' });
    expect(out).not.toContain('<img');
    expect(out).toContain('&lt;img src=x onerror=alert(1)&gt;');
  });
});
