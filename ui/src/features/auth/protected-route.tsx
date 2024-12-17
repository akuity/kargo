import { TransportProvider, useQuery } from '@connectrpc/connect-query';
import { useEffect, useRef, useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transport, transportWithAuth } from '@ui/config/transport';
import { PromotionDirectivesRegistryContextProvider } from '@ui/features/promotion-directives/registry/context/registry-context-provider';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ModalContextProvider } from '../common/modal/modal-context-provider';

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
    return <Navigate to={paths.login} replace />;
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
