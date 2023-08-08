import { useQuery } from '@tanstack/react-query';
import { Button, notification } from 'antd';
import * as oauth from 'oauth4webapi';
import React from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { OIDCConfig } from '@ui/gen/service/v1alpha1/service_pb';

const codeVerifierKey = 'PKCE_code_verifier';
const authTokenKey = 'auth_token';

type Props = {
  oidcConfig: OIDCConfig;
};

export const OIDCLogin = ({ oidcConfig }: Props) => {
  const issuer = new URL(oidcConfig.issuerUrl);
  const location = useLocation();
  const navigate = useNavigate();
  const redirectURI = `${window.location.protocol}//${window.location.hostname}:${window.location.port}${location.pathname}`;

  const client = React.useMemo(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    () => ({ client_id: oidcConfig.clientId, token_endpoint_auth_method: 'none' as any }),
    [oidcConfig]
  );

  const { data: as } = useQuery(
    [issuer],
    () =>
      oauth
        .discoveryRequest(issuer)
        .then((response) => oauth.processDiscoveryResponse(issuer, response))
        .then((response) => {
          if (response.code_challenge_methods_supported?.includes('S256') !== true) {
            throw new Error('OIDC config fetch error');
          }

          return response;
        }),
    {
      onError: (err) => {
        const errorMessage = err instanceof Error ? err.message : 'OIDC config fetch error';
        notification.error({ message: errorMessage, placement: 'bottomRight' });
      }
    }
  );

  const login = async () => {
    if (!as?.authorization_endpoint) {
      return;
    }

    const code_verifier = oauth.generateRandomCodeVerifier();
    sessionStorage.setItem(codeVerifierKey, code_verifier);

    const code_challenge = await oauth.calculatePKCECodeChallenge(code_verifier);
    const url = new URL(as.authorization_endpoint);
    url.searchParams.set('client_id', client.client_id);
    url.searchParams.set('code_challenge', code_challenge);
    url.searchParams.set('code_challenge_method', 'S256');
    url.searchParams.set('redirect_uri', redirectURI);
    url.searchParams.set('response_type', 'code');
    url.searchParams.set('scope', oidcConfig.scopes.join(' '));

    window.location.replace(url.toString());
  };

  // Handle callback from OIDC provider
  React.useEffect(() => {
    (async () => {
      const code_verifier = sessionStorage.getItem(codeVerifierKey);
      const searchParams = new URLSearchParams(location.search);

      if (!as || !code_verifier || !searchParams.get('code') || !searchParams.get('code')) {
        return;
      }

      // Delete empty state
      if (searchParams.get('state') === '') {
        searchParams.delete('state');
      }

      const params = oauth.validateAuthResponse(as, client, searchParams, oauth.expectNoState);

      if (oauth.isOAuth2Error(params)) {
        notification.error({
          message: 'OIDC: Validation Auth Response error',
          placement: 'bottomRight'
        });
        return;
      }

      const response = await oauth.authorizationCodeGrantRequest(
        as,
        client,
        params,
        redirectURI,
        code_verifier
      );

      if (oauth.parseWwwAuthenticateChallenges(response)) {
        notification.error({
          message: 'OIDC: Parsing Authenticate Challenges error',
          placement: 'bottomRight'
        });
        return;
      }

      const result = await oauth.processAuthorizationCodeOpenIDResponse(as, client, response);
      if (oauth.isOAuth2Error(result) || !result.id_token) {
        notification.error({
          message: 'OIDC: Proccess Authorization Code Grant Response error',
          placement: 'bottomRight'
        });
        return;
      }

      localStorage.setItem(authTokenKey, result.id_token);
      navigate(paths.home, { replace: true });
    })();
  }, [as, client, location]);

  return <Button onClick={login}>Login</Button>;
};
