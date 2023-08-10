import { createConnectTransport } from '@bufbuild/connect-web';

export const transport = createConnectTransport({
  baseUrl: '/api'
});
