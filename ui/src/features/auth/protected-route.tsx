import { useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { ModalContextProvider } from '@ui/features/common/modal/modal-context-provider';
import { PromotionDirectivesRegistryContextProvider } from '@ui/features/promotion-directives/registry/context/registry-context-provider';
import { useGetPublicConfig } from '@ui/gen/api/v2/system/system';

import { useAuthContext } from './context/use-auth-context';

export const ProtectedRoute = () => {
  const { isLoggedIn } = useAuthContext();
  const { data: response, isLoading } = useGetPublicConfig();
  const data = response?.data;

  const [modalRoot, setModalRoot] = useState<HTMLDivElement | null>(null);

  // Protected children can issue requests as soon as they mount. Wait until
  // skipAuth is known, so their 401s cannot race the login redirect.
  if (isLoading) {
    return null;
  }

  if (!data?.skipAuth && !isLoading && !isLoggedIn) {
    return (
      <Navigate
        to={`${paths.login}?${window.location.pathname !== paths.home ? `${redirectToQueryParam}=${window.location.pathname}` : ''}`}
        replace
      />
    );
  }

  return (
    // When we will have external runners, we should have dedicated page that shows available runners in registry
    // Its either use this context only where needed and use query caching (not a concern for now) OR just keep this available all the time without invalidation
    // Not a concern as of now but something to keep in mind
    <PromotionDirectivesRegistryContextProvider>
      <div ref={setModalRoot}>
        {modalRoot && (
          <ModalContextProvider container={modalRoot}>
            <Outlet />
          </ModalContextProvider>
        )}
      </div>
    </PromotionDirectivesRegistryContextProvider>
  );
};
