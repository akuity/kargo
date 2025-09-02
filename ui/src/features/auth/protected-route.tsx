import { TransportProvider, useQuery } from '@connectrpc/connect-query';
import { useEffect, useRef, useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { transport, transportWithAuth } from '@ui/config/transport';
import { ModalContextProvider } from '@ui/features/common/modal/modal-context-provider';
import { PromotionDirectivesRegistryContextProvider } from '@ui/features/promotion-directives/registry/context/registry-context-provider';
import { getPublicConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import { useAuthContext } from './context/use-auth-context';

export const ProtectedRoute = () => {
  const { isLoggedIn } = useAuthContext();
  const { data, isLoading } = useQuery(getPublicConfig);

  const modalRef = useRef<HTMLDivElement>(null);
  const [modalRoot, setModalRoot] = useState<HTMLDivElement | null>(null);
  useEffect(() => {
    setModalRoot(modalRef.current);
  }, []);

  if (!data?.skipAuth && !isLoading && !isLoggedIn) {
    return (
      <Navigate
        to={`${paths.login}?${window.location.pathname !== paths.home ? `${redirectToQueryParam}=${window.location.pathname}` : ''}`}
        replace
      />
    );
  }

  return (
    <TransportProvider transport={data?.skipAuth ? transport : transportWithAuth}>
      {/* 
        When we will have external runners, we should have dedicated page that shows available runners in registry
        Its either use this context only where needed and use query caching (not a concern for now) OR just keep this available all the time without invalidation
        Not a concern as of now but something to keep in mind
      */}
      <PromotionDirectivesRegistryContextProvider>
        <div ref={modalRef}>
          {modalRoot && (
            <ModalContextProvider container={modalRoot}>
              <Outlet />
            </ModalContextProvider>
          )}
        </div>
      </PromotionDirectivesRegistryContextProvider>
    </TransportProvider>
  );
};
