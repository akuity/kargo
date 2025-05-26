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

export const claimsMapping: Record<string, { label: string; description: string }> = {
  iss: {
    label: 'Issuer',
    description: 'Identifies the principal that issued the JWT.'
  },
  sub: {
    label: 'Subject',
    description: 'Identifies the principal that is the subject of the JWT'
  },
  aud: {
    label: 'Audience',
    description: 'Identifies the recipients the JWT is intended for (can be a string or array).'
  },
  exp: {
    label: 'Expiration Time',
    description:
      'Token expiration time (in Unix timestamp). After this time, the token must not be accepted.'
  },
  iat: {
    label: 'Issued At',
    description: 'The time the JWT was issued (in Unix timestamp).'
  },
  jti: {
    label: 'JWT ID',
    description: 'Unique identifier for the JWT, used to prevent replay attacks.'
  }
};
