import { ConnectError, Interceptor } from '@bufbuild/connect';
import { createConnectTransport } from '@bufbuild/connect-web';
import { notification } from 'antd';

const errorHandler: Interceptor = (next) => async (req) => {
  try {
    return await next(req);
  } catch (err) {
    if (err instanceof ConnectError) {
      notification.error({ message: err.rawMessage, placement: 'bottomRight' });
    }

    throw err;
  }
};

export const transport = createConnectTransport({
  baseUrl: '',
  interceptors: [errorHandler]
});
