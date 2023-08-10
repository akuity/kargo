import { useQuery } from '@tanstack/react-query';
import { Divider, Typography } from 'antd';

import { AdminLogin } from '@ui/features/auth/admin-login';
import { OIDCLogin } from '@ui/features/auth/oidc-login';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import * as styles from './login.module.less';

export const Login = () => {
  const { data } = useQuery(getPublicConfig.useQuery());

  return (
    <div className={styles.container}>
      <div className={styles.logo}>
        <img src='/kargo-icon.png' alt='Kargo Icon' className={styles.icon} />
        Kargo
      </div>
      <div className={styles.box}>
        <Typography.Title level={4}>Admin Login</Typography.Title>
        <AdminLogin />
        {data?.oidcConfig && (
          <>
            <Divider className='!my-6 !text-gray-400 !font-light'>OR</Divider>
            <OIDCLogin oidcConfig={data.oidcConfig} />
          </>
        )}
      </div>
    </div>
  );
};
