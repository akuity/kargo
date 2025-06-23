import { useQuery } from '@connectrpc/connect-query';
import { PropsWithChildren } from 'react';

import { getConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { ConfigContext } from './config-context';

export const ConfigContextProvider = (props: PropsWithChildren<object>) => {
  const getConfigQuery = useQuery(getConfig);

  return (
    <ConfigContext.Provider
      value={{ config: getConfigQuery.data, isFetching: getConfigQuery.isFetching }}
    >
      {props.children}
    </ConfigContext.Provider>
  );
};
