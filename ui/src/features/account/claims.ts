import { format, formatDistance } from 'date-fns';

import { asText, isAdmin, JWTInfo } from '@ui/features/auth/jwt-utils';
import type { PublicConfig } from '@ui/gen/api/v2/models';

type ClaimInfo = { label: string; description: string };

// The claims the readable view can explain, in the order it lists them: where
// the token came from and who it is about, then the times it holds. Any other
// claim follows these, by name. A Map rather than an object, so a claim named
// after something on Object.prototype is not mistaken for a known one.
const knownClaims = new Map<string, ClaimInfo>([
  ['iss', { label: 'Issuer', description: 'Who issued this token' }],
  ['aud', { label: 'Audience', description: 'Who the token is intended for' }],
  ['azp', { label: 'Authorized party', description: 'The client the token was issued to' }],
  ['sub', { label: 'Subject', description: 'Your user identifier' }],
  ['name', { label: 'Name', description: 'Your full name' }],
  ['preferred_username', { label: 'Username', description: 'The name you prefer to go by' }],
  ['email', { label: 'Email', description: 'Your email address' }],
  [
    'email_verified',
    { label: 'Email verified', description: 'Whether your identity provider verified it' }
  ],
  ['groups', { label: 'Groups', description: 'Groups you belong to, used to map you to roles' }],
  ['iat', { label: 'Issued at', description: 'When the token was created' }],
  ['auth_time', { label: 'Signed in at', description: 'When you last signed in' }],
  ['nbf', { label: 'Not before', description: 'Token is not valid before this time' }],
  ['exp', { label: 'Expires', description: 'Token stops working after this time' }],
  ['jti', { label: 'Token ID', description: 'Unique ID of this token' }]
]);

const claimOrder = [...knownClaims.keys()];

// Claims that hold a time, as seconds since the epoch (RFC 7519's NumericDate).
const timeClaims = new Set(['iat', 'auth_time', 'nbf', 'exp']);

export type ClaimValue =
  | { kind: 'time'; date: Date; seconds: number }
  | { kind: 'list'; items: string[] }
  | { kind: 'text'; text: string };

export type Claim = {
  key: string;
  label: string;
  description?: string;
  value: ClaimValue;
  // What copying the claim puts on the clipboard: the value as the token has
  // it, so a time copies as its Unix seconds rather than as it is shown.
  copy: string;
};

type Primitive = string | number | boolean;

const isPrimitive = (value: unknown): value is Primitive =>
  typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean';

// A claim as plain text: strings as they are, anything else as JSON.
export const claimText = (raw: unknown) => {
  if (typeof raw === 'string') {
    return raw;
  }
  return JSON.stringify(raw) ?? '';
};

// The identity provider can put anything in a claim, so each is read for what
// it holds rather than for what the claim is meant to hold.
export const claimValue = (key: string, raw: unknown): ClaimValue => {
  if (timeClaims.has(key) && typeof raw === 'number') {
    const date = new Date(raw * 1000);
    if (!isNaN(date.getTime())) {
      return { kind: 'time', date, seconds: raw };
    }
  }
  if (Array.isArray(raw) && raw.every(isPrimitive)) {
    return { kind: 'list', items: raw.map(String) };
  }
  return { kind: 'text', text: claimText(raw) };
};

const claimRank = (key: string) => {
  const index = claimOrder.indexOf(key);
  return index === -1 ? claimOrder.length : index;
};

export const claimRows = (claims: Record<string, unknown>): Claim[] =>
  Object.keys(claims)
    .sort((a, b) => claimRank(a) - claimRank(b) || a.localeCompare(b))
    .map((key) => ({
      key,
      label: knownClaims.get(key)?.label ?? key,
      description: knownClaims.get(key)?.description,
      value: claimValue(key, claims[key]),
      copy: claimText(claims[key])
    }));

// The offset goes with the time: tokens are compared against more than one
// clock when something goes wrong, so the zone it is shown in matters.
export const formatClaimTime = (date: Date) => format(date, 'EEE, MMM d yyyy, HH:mm:ss O');

export const relativeTime = (date: Date, now: Date) =>
  formatDistance(date, now, { addSuffix: true });

// Whether the claim says the token has stopped working by `now`.
export const claimExpired = (claim: Claim, now: Date) =>
  claim.key === 'exp' && claim.value.kind === 'time' && claim.value.date <= now;

// One slash is all issuers differ by in practice. No regex: `/\/+$/` backtracks
// quadratically on a long run of slashes that is not at the end.
const withoutTrailingSlash = (url: string) => (url.endsWith('/') ? url.slice(0, -1) : url);

// Whether the token is the one the API server issues its built-in admin
// account. The server knows that token by its issuer, which the UI is not
// told; what it is told is whether there is an admin account at all, and
// which issuer single sign-on uses. A subject of "admin" counts only where
// both allow it, so an identity provider that calls a user "admin" is not
// taken for it. Until the config is in, the subject alone decides.
export const isBuiltInAdmin = (jwt?: JWTInfo | null, config?: PublicConfig) => {
  if (!isAdmin(jwt)) {
    return false;
  }
  if (!config) {
    return true;
  }
  if (!config.adminAccountEnabled) {
    return false;
  }
  const ssoIssuer = asText(config.oidcConfig?.issuerUrl);
  return !ssoIssuer || withoutTrailingSlash(asText(jwt?.iss)) !== withoutTrailingSlash(ssoIssuer);
};

// The issuer's host, where it is a URL with one. `new URL` takes any `word:`
// prefix for a scheme, so `localhost:5556` parses too -- with no host.
const issuerHost = (issuer: string) => {
  try {
    return new URL(issuer).host || issuer;
  } catch {
    return issuer;
  }
};

// How the token was obtained: the built-in admin login, or single sign-on
// through the issuer, named by its host where it is a URL.
export const signInMethod = (jwt: JWTInfo | null | undefined, admin: boolean) => {
  if (admin) {
    return 'built-in admin login';
  }
  const issuer = asText(jwt?.iss);
  return issuer ? `single sign-on (${issuerHost(issuer)})` : 'single sign-on';
};
