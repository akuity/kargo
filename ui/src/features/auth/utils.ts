// jwt claims register with "admin" subject on admin login
export const isAdmin = (user: JWTUserInfo) => user.sub === 'admin';

// https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
export type JWTUserInfo = {
  sub: string;
  iss: string;
  groups?: string[];
  name?: string;
  preferred_username?: string;
  email?: string;
};

export const extractUserInfoFromJWT = (token: string): JWTUserInfo => {};
