import React, { PropsWithChildren } from 'react';

import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { useGetVersionInfo } from '@ui/gen/api/v2/system/system';

import { ProductEdition, ProductEditionContext } from './product-edition-context';

export const ProductEditionContextProvider = ({ children }: PropsWithChildren) => {
  const { isLoggedIn } = useAuthContext();
  const { data, isLoading } = useGetVersionInfo({
    query: { enabled: isLoggedIn }
  });
  const edition =
    data?.data.edition === ProductEdition.Enterprise
      ? ProductEdition.Enterprise
      : ProductEdition.Community;
  const context = React.useMemo(
    () => ({
      edition,
      isEnterprise: edition === ProductEdition.Enterprise,
      isLoading
    }),
    [edition, isLoading]
  );

  return (
    <ProductEditionContext.Provider value={context}>{children}</ProductEditionContext.Provider>
  );
};
