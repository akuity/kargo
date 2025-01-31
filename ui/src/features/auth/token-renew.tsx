import { useQuery as useConnectQuery } from '@connectrpc/connect-query';
import { useQuery } from '@tanstack/react-query';
import { notification } from 'antd';
import {
  allowInsecureRequests,
  discoveryRequest,
  processDiscoveryResponse,
  refreshTokenGrantRequest,
  processRefreshTokenResponse
} from 'oauth4webapi';
import React from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { redirectToQueryParam, refreshTokenKey } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { getPublicConfig } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';

import { LoadingState } from '../common';

import { useAuthContext } from './context/use-auth-context';
import { oidcClientAuth, shouldAllowIdpHttpRequest as shouldAllowHttpRequest } from './oidc-utils';

export const TokenRenew = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { login: onLogin, logout } = useAuthContext();

  const { data, isError } = useConnectQuery(getPublicConfig);

  const issuerUrl = React.useMemo(() => {
    try {
      return data?.oidcConfig?.issuerUrl ? new URL(data?.oidcConfig?.issuerUrl) : undefined;
    } catch (_) {
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
      discoveryRequest(issuerUrl, {
        [allowInsecureRequests]: shouldAllowHttpRequest()
      })
        .then((response) => processDiscoveryResponse(issuerUrl, response))
        .then((response) => {
          if (
            response.code_challenge_methods_supported?.includes('S256') !== true &&
            !issuerUrl.toString().startsWith('https://login.microsoftonline.com')
          ) {
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
      return;
    }

    (async () => {
      try {
        const response = await refreshTokenGrantRequest(as, client, oidcClientAuth, refreshToken, {
          [allowInsecureRequests]: shouldAllowHttpRequest(),
          additionalParameters: [['client_id', client.client_id]]
        });

        const result = await processRefreshTokenResponse(as, client, response);

        if (!result.id_token) {
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
      } catch (err) {
        notification.error({
          message: `OIDC: ${JSON.stringify(err)}`,
          placement: 'bottomRight'
        });

        logout();
        navigate(paths.login);
      }
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
