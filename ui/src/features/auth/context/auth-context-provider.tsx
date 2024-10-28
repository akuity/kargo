import { useQuery } from '@connectrpc/connect-query';
import React, { PropsWithChildren, useEffect } from 'react';

import { authTokenKey, refreshTokenKey } from '@ui/config/auth';
import { transportWithAuth } from '@ui/config/transport';
import { LoadingState } from '@ui/features/common';
import { whoAmI } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { AuthContext } from './auth-context';

export const AuthContextProvider = ({ children }: PropsWithChildren) => {
  const [token, setToken] = React.useState(localStorage.getItem(authTokenKey));

  const whoAmIQuery = useQuery(whoAmI, undefined, {
    transport: transportWithAuth
  });

  useEffect(() => {
    whoAmIQuery.refetch();
  }, [token]);

  // who-am-i call succeeds only when user has valid token
  const isLoggedIn = !!token && !whoAmIQuery.isError;

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
      isLoggedIn,
      login,
      logout
    }),
    [login, logout, isLoggedIn, token]
  );

  if (whoAmIQuery.isFetching) {
    return <LoadingState />;
  }

  return <AuthContext.Provider value={ctx}>{children}</AuthContext.Provider>;
};
