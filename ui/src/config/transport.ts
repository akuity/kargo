import { Code, ConnectError, Interceptor } from '@bufbuild/connect';
import { createConnectTransport } from '@bufbuild/connect-web';
import { notification } from 'antd';

import { paths } from './paths';

export const authTokenKey = 'auth_token';

const errorHandler: Interceptor = (next) => async (req) => {
  try {
    const token = localStorage.getItem(authTokenKey);

    if (token) {
      req.header.append('Authorization', `Bearer ${token}`);
    }

    return await next(req);
  } catch (err) {
    if (req.signal.aborted) {
      throw err;
    }

    const errorMessage = err instanceof ConnectError ? err.rawMessage : 'Unexpected API error';
    notification.error({ message: errorMessage, placement: 'bottomRight' });

    if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
      localStorage.removeItem(authTokenKey);

      setTimeout(() => window.location.replace(paths.login), 3000);
    }

    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl: '',
  interceptors: [errorHandler]
});
