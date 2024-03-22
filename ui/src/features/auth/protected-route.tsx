import { TransportProvider } from '@connectrpc/connect-query';
import { useEffect, useRef, useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';

import { ModalContextProvider } from '../common/modal/modal-context';

import { useAuthContext } from './context/use-auth-context';

export const ProtectedRoute = () => {
  const { isLoggedIn } = useAuthContext();

  if (!isLoggedIn) {
    return <Navigate to={paths.login} replace />;
  }

  const modalRef = useRef<HTMLDivElement>(null);
  const [modalRoot, setModalRoot] = useState<HTMLDivElement | null>(null);
  useEffect(() => {
    setModalRoot(modalRef.current);
  }, []);

  return (
    <TransportProvider transport={transportWithAuth}>
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
