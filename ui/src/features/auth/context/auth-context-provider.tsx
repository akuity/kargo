import React, { PropsWithChildren } from 'react';

import { authTokenKey } from '@ui/config/transport';

import { AuthContext } from './auth-context';

export const AuthContextProvider = ({ children }: PropsWithChildren) => {
  const [token, setToken] = React.useState(localStorage.getItem(authTokenKey));

  const login = React.useCallback((t: string) => {
    localStorage.setItem(authTokenKey, t);
    setToken(t);
  }, []);

  const logout = React.useCallback(() => {
    localStorage.removeItem(authTokenKey);
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
