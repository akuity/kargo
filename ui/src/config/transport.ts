import { Code, ConnectError, Interceptor } from '@bufbuild/connect';
import { createConnectTransport } from '@bufbuild/connect-web';
import { notification } from 'antd';

import { paths } from './paths';

export const authTokenKey = 'auth_token';

const logout = () => {
  localStorage.removeItem(authTokenKey);
  window.location.replace(paths.login);
};

const authHandler: Interceptor = (next) => (req) => {
  const token = localStorage.getItem(authTokenKey);
  let isTokenExpired;

  try {
    isTokenExpired = token && Date.now() >= JSON.parse(atob(token.split('.')[1])).exp * 1000;
  } catch (err) {
    logout();

    throw new ConnectError('Invalid token');
  }

  if (isTokenExpired) {
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
  interceptors: [authHandler, errorHandler]
});
