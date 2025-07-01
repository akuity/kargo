import { useQuery } from '@connectrpc/connect-query';
import { Divider, Typography } from 'antd';
import { Navigate, generatePath, useSearchParams } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { AdminLogin } from '@ui/features/auth/admin-login';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { OIDCLogin } from '@ui/features/auth/oidc-login';
import { LoadingState } from '@ui/features/common';
import { KargoLogo } from '@ui/features/common/logo/logo';
import { getPublicConfig } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';

import * as styles from './login.module.less';

export const Login = () => {
  const { data, isLoading } = useQuery(getPublicConfig);
  const [params] = useSearchParams();
  const { isLoggedIn } = useAuthContext();
  const redirectTo = params.get(redirectToQueryParam);

  if (data?.skipAuth) {
    return <Navigate to={paths.home} replace />;
  }

  if (isLoggedIn) {
    return <Navigate to={redirectTo ? generatePath(redirectTo) : paths.home} replace />;
  }

  return (
    <div className={styles.container}>
      <KargoLogo />
      <div className={styles.box}>
        {isLoading && <LoadingState />}
        {data?.adminAccountEnabled && (
          <>
            <Typography.Title level={4}>Admin Login</Typography.Title>
            <AdminLogin />
          </>
        )}
        {data?.oidcConfig && data?.adminAccountEnabled && (
          <Divider className='!my-6 !text-gray-400 !font-light'>OR</Divider>
        )}
        {data?.oidcConfig && <OIDCLogin oidcConfig={data.oidcConfig} />}
        {data && !data.oidcConfig && !data.adminAccountEnabled && (
          <Typography.Text>
            Login is disabled. Please contact your system administrator.
          </Typography.Text>
        )}
      </div>
    </div>
  );
};
