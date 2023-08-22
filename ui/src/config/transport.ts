import { ConnectError, Interceptor } from '@bufbuild/connect';
import { createConnectTransport } from '@bufbuild/connect-web';
import { notification } from 'antd';

const errorHandler: Interceptor = (next) => async (req) => {
  try {
    return await next(req);
  } catch (err) {
    if (!req.signal.aborted) {
      const errorMessage = err instanceof ConnectError ? err.rawMessage : 'Unexpected API error';
      notification.error({ message: errorMessage, placement: 'bottomRight' });
    }

    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl: '',
  interceptors: [errorHandler]
});
