import React, { PropsWithChildren } from 'react';

import { authTokenKey, refreshTokenKey } from '@ui/config/auth';

import { AuthContext } from './auth-context';

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

  const ctx = React.useMemo(
    () => ({
      isLoggedIn: !!token,
      login,
      logout
    }),
    [login, logout, token]
  );

  return <AuthContext.Provider value={ctx}>{children}</AuthContext.Provider>;
};
