import { TransportProvider, useQuery } from '@connectrpc/connect-query';
import { useEffect, useRef, useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transport, transportWithAuth } from '@ui/config/transport';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { ModalContextProvider } from '../common/modal/modal-context';

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
      <div ref={modalRef}>
        {modalRoot && (
          <ModalContextProvider container={modalRoot}>
            <Outlet />
          </ModalContextProvider>
        )}
      </div>
    </TransportProvider>
  );
};
