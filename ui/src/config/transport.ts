import { createConnectTransport } from '@bufbuild/connect-web';

export const transport = createConnectTransport({
  baseUrl: 'http://localhost:3333/api' // TODO: env variable
});
