import { Code } from '@connectrpc/connect';

import { defaultErrorHandler, newErrorHandler, newTransportWithAuth } from '@ui/config/transport';

export const clusterConfigTransport = newTransportWithAuth(
  newErrorHandler((err) => {
    if (err.code === Code.NotFound) {
      // ignore 404 because ClusterConfig may not be created
      // and we need to ignore the error
      return;
    }

    defaultErrorHandler(err);
  })
);
