import { TransportProvider } from '@bufbuild/connect-query';
import { Navigate, Outlet } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';

import { useAuthContext } from './context/use-auth-context';

export const ProtectedRoute = () => {
  const { isLoggedIn } = useAuthContext();

  if (!isLoggedIn) {
    return <Navigate to={paths.login} replace />;
  }

  return (
    <TransportProvider transport={transportWithAuth}>
      <Outlet />
    </TransportProvider>
  );
};
