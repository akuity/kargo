import { useQuery as useConnectQuery } from '@connectrpc/connect-query';
import { useQuery } from '@tanstack/react-query';
import { notification } from 'antd';
import * as oauth from 'oauth4webapi';
import React from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { redirectToQueryParam, refreshTokenKey } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { LoadingState } from '../common';

import { useAuthContext } from './context/use-auth-context';

export const TokenRenew = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login: onLogin, logout } = useAuthContext();

  const { data, isError } = useConnectQuery(getPublicConfig);

  const issuerUrl = React.useMemo(() => {
    try {
      return data?.oidcConfig?.issuerUrl ? new URL(data?.oidcConfig?.issuerUrl) : undefined;
    } catch (err) {
      notification.error({
        message: 'Invalid issuerURL',
        placement: 'bottomRight'
      });
    }
  }, [data?.oidcConfig?.issuerUrl]);

  const client = React.useMemo(
    () =>
      data?.oidcConfig?.clientId
        ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
          { client_id: data?.oidcConfig?.clientId, token_endpoint_auth_method: 'none' as any }
        : undefined,
    [data?.oidcConfig?.clientId]
  );

  const { data: as, isError: isASError } = useQuery({
    queryKey: [issuerUrl],
    queryFn: () =>
      issuerUrl &&
      oauth
        .discoveryRequest(issuerUrl)
        .then((response) => oauth.processDiscoveryResponse(issuerUrl, response))
        .then((response) => {
          if (response.code_challenge_methods_supported?.includes('S256') !== true) {
            throw new Error('OIDC config fetch error');
          }

          return response;
        }),
    enabled: !!issuerUrl
  });

  React.useEffect(() => {
    const refreshToken = localStorage.getItem(refreshTokenKey);

    if (!refreshToken) {
      navigate(paths.home);

      return;
    }

    if (!as || !client) {
      navigate(paths.login);

      return;
    }

    (async () => {
      const response = await oauth.refreshTokenGrantRequest(as, client, refreshToken);

      const result = await oauth.processRefreshTokenResponse(as, client, response);
      if (oauth.isOAuth2Error(result) || !result.id_token) {
        notification.error({
          message: 'OIDC: Proccess Authorization Code Grant Response error',
          placement: 'bottomRight'
        });
        logout();
        navigate(paths.login);
        return;
      }

      onLogin(result.id_token, result.refresh_token);
      navigate(searchParams.get(redirectToQueryParam) || paths.home);
    })();
  }, [as, client]);

  React.useEffect(() => {
    if (isError || isASError) {
      logout();
      navigate(paths.login);
    }
  }, [isError, isASError]);

  return (
    <div className='pt-40'>
      <LoadingState />
    </div>
  );
};
