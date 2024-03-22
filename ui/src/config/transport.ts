import { Code, ConnectError, Interceptor } from '@connectrpc/connect';
import { createConnectTransport } from '@connectrpc/connect-web';
import { notification } from 'antd';

import { authTokenKey, redirectToQueryParam, refreshTokenKey } from './auth';
import { paths } from './paths';

const logout = () => {
  localStorage.removeItem(authTokenKey);
  window.location.replace(paths.login);
};

const renewToken = () => {
  window.location.replace(
    `${paths.tokenRenew}?${redirectToQueryParam}=${window.location.pathname}`
  );
};

const authHandler: Interceptor = (next) => async (req) => {
  const token = localStorage.getItem(authTokenKey);
  const refreshToken = localStorage.getItem(refreshTokenKey);
  let isTokenExpired;

  try {
    isTokenExpired = token && Date.now() >= JSON.parse(atob(token.split('.')[1])).exp * 1000;
  } catch (err) {
    logout();

    throw new ConnectError('Invalid token');
  }

  if (isTokenExpired && refreshToken) {
    renewToken();
    throw new ConnectError('Token expired');
  }

  if (isTokenExpired && !refreshToken) {
    logout();
    throw new ConnectError('Token expired');
  }

  if (token) {
    req.header.append('Authorization', `Bearer ${token}`);
  }

  return next(req);
};

const errorHandler: Interceptor = (next) => (req) => {
  try {
    return next(req);
  } catch (err) {
    if (req.signal.aborted) {
      throw err;
    }

    const errorMessage = err instanceof ConnectError ? err.rawMessage : 'Unexpected API error';
    notification.error({ message: errorMessage, placement: 'bottomRight' });

    if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
      logout();
    }

    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl: '',
  useBinaryFormat: true,
  interceptors: [errorHandler]
});

export const transportWithAuth = createConnectTransport({
  baseUrl: '',
  useBinaryFormat: true,
  interceptors: [authHandler, errorHandler]
});
