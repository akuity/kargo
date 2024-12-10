import React, { PropsWithChildren, useMemo } from 'react';

import { authTokenKey, refreshTokenKey } from '@ui/config/auth';

import { extractInfoFromJWT, JWTInfo } from '../jwt-utils';

import { AuthContext, AuthContextType } from './auth-context';

export const AuthContextProvider = ({ children }: PropsWithChildren) => {
  const [token, setToken] = React.useState(localStorage.getItem(authTokenKey));

  const login = React.useCallback((token: string, refreshToken?: string) => {
    localStorage.setItem(authTokenKey, token);

    if (refreshToken) {
      localStorage.setItem(refreshTokenKey, refreshToken);
    }

    setToken(token);
  }, []);

  const logout = React.useCallback(() => {
    localStorage.removeItem(authTokenKey);
    localStorage.removeItem(refreshTokenKey);

    setToken(null);
  }, []);

  const jwtInfo: JWTInfo | null = useMemo(() => {
    if (token) {
      try {
        return extractInfoFromJWT(token);
      } catch {
        // if "something" is off with token (assume isLoggedIn is true because it just check whether token is present or not ie. not validity of token)
        // authHandler interceptor will find before any API call and redirect to login page anyways
        // consumer will decide whats the best UX at that point
        return null;
      }
    }

    return null;
  }, [token]);

  const ctx: AuthContextType = React.useMemo(
    () => ({
      isLoggedIn: !!token,
      login,
      logout,
      JWTInfo: jwtInfo
    }),
    [login, logout, token, jwtInfo]
  );

  return <AuthContext.Provider value={ctx}>{children}</AuthContext.Provider>;
};
