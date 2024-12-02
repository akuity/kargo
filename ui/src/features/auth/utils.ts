import { ClientAuth } from 'oauth4webapi';

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

export const extractInfoFromJWT = (token: string): JWTInfo => JSON.parse(atob(token.split('.')[1]));

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

export const oidcClientAuth: ClientAuth = () => {
  // equivalent function for token_endpoint_auth_method: 'none'
};

export const shouldAllowIdpHttpRequest = () => __UI_VERSION__ === 'development';
