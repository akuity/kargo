import { Code } from '@connectrpc/connect';

import { defaultErrorHandler, newErrorHandler, newTransportWithAuth } from '@ui/config/transport';

export const projectConfigTransport = newTransportWithAuth(
  newErrorHandler((err) => {
    if (err.code === Code.NotFound) {
      // ignore 404 because ProjectConfig may not be created when we create a new Project
      // and we need to ignore the error
      return;
    }

    defaultErrorHandler(err);
  })
);
