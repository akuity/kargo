import { useQuery } from '@tanstack/react-query';

import { OIDCLogin } from '@ui/features/auth/oidc-login';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

export const Login = () => {
  const { data } = useQuery(getPublicConfig.useQuery());

  return <>{data?.oidcConfig && <OIDCLogin oidcConfig={data.oidcConfig} />}</>;
};
