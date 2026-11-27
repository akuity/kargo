import { Flex, Typography } from 'antd';
import { Navigate } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { withoutBasePath } from '@ui/config/base-path';
import { paths } from '@ui/config/paths';
import { AccountCard } from '@ui/features/account/account-card';
import { TokenClaimsCard } from '@ui/features/account/token-claims-card';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { isJWTDirty } from '@ui/features/auth/jwt-utils';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';

export const User = () => {
  useDocumentTitle(['Your account']);
  const { JWTInfo } = useAuthContext();

  if (isJWTDirty(JWTInfo)) {
    // other pages depend on API failure code to redirect to login
    // since this page makes no authenticated API call, this is the only reason to redirect
    return (
      <Navigate
        to={`${paths.login}?${redirectToQueryParam}=${withoutBasePath(window.location.pathname)}`}
      />
    );
  }

  return (
    <div className='p-6'>
      {/* As wide as Settings with its navigation: the 240px menu, the 24px gap
          and the 920px content column (see pages/settings.tsx). */}
      <div className='max-w-[1184px]'>
        <Typography.Title className='!mb-1'>Your account</Typography.Title>
        <Typography.Paragraph type='secondary' className='!mb-6'>
          Who you&apos;re signed in as, and the token Kargo is using on your behalf.
        </Typography.Paragraph>
        <Flex vertical gap={16}>
          <AccountCard />
          {JWTInfo && <TokenClaimsCard claims={JWTInfo} />}
        </Flex>
      </div>
    </div>
  );
};
