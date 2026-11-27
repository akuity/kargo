import { parseJwtPayload } from '@ui/utils/jwt-payload';

// https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
export type JWTInfo = {
  sub: string;
  iss: string;
  groups?: string[];
  name?: string;
  preferred_username?: string;
  email?: string;
};

// jwt claims register with "admin" subject on admin login
export const isAdmin = (user?: JWTInfo | null) => user?.sub === 'admin';

export const extractInfoFromJWT = (token: string): JWTInfo => parseJwtPayload<JWTInfo>(token);

export const isJWTValid = (token: string): boolean => {
  try {
    extractInfoFromJWT(token);
    return true;
  } catch {
    return false;
  }
};

// rare but possible case if jwt token does not have valid information UI has to take some decisions
// for example redirect to login page or not, show the users page in sidebar or not
// this inherits the information that user is logged-in already ie. local-storage has token
export const isJWTDirty = (jwt?: JWTInfo | null) => jwt === null;

export const getUserEmail = (user?: JWTInfo | null) => {
  let meta = '';

  if (isAdmin(user)) {
    meta = 'Admin';
  } else if (user?.email) {
    meta = user.email;
  }

  return meta;
};

// Claims come straight from the identity provider's token, so despite JWTInfo's
// types they may be numbers or objects. Only non-blank strings are shown.
export const asText = (value: unknown) => (typeof value === 'string' ? value.trim() : '');

export const getDisplayName = (jwt?: JWTInfo | null) =>
  asText(jwt?.name) ||
  asText(jwt?.preferred_username) ||
  asText(jwt?.email) ||
  asText(jwt?.sub) ||
  'Account';
