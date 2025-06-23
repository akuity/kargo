import { createContext } from 'react';

import { GetConfigResponse } from '@ui/gen/api/service/v1alpha1/service_pb';

export interface ConfigContextType {
  config: GetConfigResponse | undefined;
  isFetching?: boolean;
}

export const ConfigContext = createContext<ConfigContextType | null>(null);
