import { useQuery } from '@tanstack/react-query';
import { Button, notification } from 'antd';
import * as oauth from 'oauth4webapi';
import React from 'react';
import { useLocation } from 'react-router-dom';

import { OIDCConfig } from '@ui/gen/service/v1alpha1/service_pb';

import { useAuthContext } from './context/use-auth-context';

const codeVerifierKey = 'PKCE_code_verifier';

type Props = {
  oidcConfig: OIDCConfig;
};

export const OIDCLogin = ({ oidcConfig }: Props) => {
  const location = useLocation();
  const redirectURI = window.location.origin + window.location.pathname;
  const { login: onLogin } = useAuthContext();

  const issuerUrl = React.useMemo(() => {
    try {
      return new URL(oidcConfig.issuerUrl);
    } catch (err) {
      notification.error({
        message: 'Invalid issuerURL',
        placement: 'bottomRight'
      });
    }
  }, [oidcConfig.issuerUrl]);

  const client = React.useMemo(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    () => ({ client_id: oidcConfig.clientId, token_endpoint_auth_method: 'none' as any }),
    [oidcConfig]
  );

  const { data: as, isFetching } = useQuery(
    [issuerUrl],
    () =>
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
    {
      enabled: !!issuerUrl,
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

      onLogin(result.id_token);
    })();
  }, [as, client, location]);

  return (
    <Button onClick={login} block size='large' loading={isFetching} disabled={!issuerUrl}>
      SSO Login
    </Button>
  );
};
