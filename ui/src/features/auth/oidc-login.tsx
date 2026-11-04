import { useQuery } from '@tanstack/react-query';
import { Button, notification } from 'antd';
import {
  discoveryRequest,
  processDiscoveryResponse,
  generateRandomCodeVerifier,
  generateRandomNonce,
  generateRandomState,
  calculatePKCECodeChallenge,
  validateAuthResponse,
  authorizationCodeGrantRequest,
  processAuthorizationCodeResponse,
  AuthorizationResponseError,
  WWWAuthenticateChallengeError,
  allowInsecureRequests
} from 'oauth4webapi';
import React from 'react';
import { useLocation } from 'react-router-dom';

import { isSafeRedirectPath, redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { OIDCConfig } from '@ui/gen/api/v2/models';

import { useAuthContext } from './context/use-auth-context';
import {
  consumeOIDCCallbackArtifacts,
  getOIDCScopes,
  oidcClientAuth,
  oidcErrorMessage,
  oidcSessionStorageKeys,
  shouldAllowIdpHttpRequest as shouldAllowHttpRequest
} from './oidc-utils';

type Props = {
  oidcConfig: OIDCConfig;
};

export const OIDCLogin = ({ oidcConfig }: Props) => {
  const location = useLocation();
  const redirectURI = window.location.origin + window.location.pathname;
  const { login: onLogin } = useAuthContext();

  const issuerUrl = React.useMemo(() => {
    try {
      return oidcConfig.issuerUrl ? new URL(oidcConfig.issuerUrl) : undefined;
    } catch (_) {
      notification.error({
        message: 'Invalid issuerURL',
        placement: 'bottomRight'
      });
    }
  }, [oidcConfig.issuerUrl]);

  const client = React.useMemo(
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    () => ({ client_id: oidcConfig.clientId ?? '', token_endpoint_auth_method: 'none' as any }),
    [oidcConfig]
  );

  const {
    data: as,
    isFetching,
    error
  } = useQuery({
    queryKey: [issuerUrl],
    queryFn: () =>
      issuerUrl &&
      discoveryRequest(issuerUrl, {
        [allowInsecureRequests]: shouldAllowHttpRequest()
      }).then((response) => processDiscoveryResponse(issuerUrl, response)),
    enabled: !!issuerUrl
  });

  React.useEffect(() => {
    if (error) {
      const errorMessage = error instanceof Error ? error.message : 'OIDC config fetch error';
      notification.error({ message: `OIDC: ${errorMessage}`, placement: 'bottomRight' });
    }
  }, [error]);

  const login = async () => {
    if (!as?.authorization_endpoint) {
      return;
    }

    const code_verifier = generateRandomCodeVerifier();
    sessionStorage.setItem(oidcSessionStorageKeys.codeVerifier, code_verifier);
    const state = generateRandomState();
    sessionStorage.setItem(oidcSessionStorageKeys.state, state);
    const nonce = generateRandomNonce();
    sessionStorage.setItem(oidcSessionStorageKeys.nonce, nonce);
    sessionStorage.setItem(oidcSessionStorageKeys.platformRedirect, window.location.search);

    const code_challenge = await calculatePKCECodeChallenge(code_verifier);
    const url = new URL(as.authorization_endpoint);
    url.searchParams.set('client_id', client.client_id);
    url.searchParams.set('code_challenge', code_challenge);
    url.searchParams.set('code_challenge_method', 'S256');
    url.searchParams.set('redirect_uri', redirectURI);
    url.searchParams.set('response_type', 'code');
    url.searchParams.set('scope', getOIDCScopes(oidcConfig, as).join(' '));
    url.searchParams.set('state', state);
    url.searchParams.set('nonce', nonce);

    window.location.replace(url.toString());
  };

  // Handle callback from OIDC provider
  React.useEffect(() => {
    (async () => {
      const code_verifier = sessionStorage.getItem(oidcSessionStorageKeys.codeVerifier);
      const state = sessionStorage.getItem(oidcSessionStorageKeys.state);
      const nonce = sessionStorage.getItem(oidcSessionStorageKeys.nonce);
      const platformRedirect = sessionStorage.getItem(oidcSessionStorageKeys.platformRedirect);
      const searchParams = new URLSearchParams(location.search);
      const isOIDCCallback = searchParams.has('code') || searchParams.has('error');

      if (!as || !isOIDCCallback || !searchParams.get('state')) {
        return;
      }

      consumeOIDCCallbackArtifacts(sessionStorage, new URL(window.location.href), (url) =>
        window.history.replaceState(window.history.state, '', url)
      );

      if (!code_verifier || !state || !nonce) {
        notification.error({
          message: 'OIDC: Login state missing or expired',
          placement: 'bottomRight'
        });
        return;
      }

      try {
        const params = validateAuthResponse(as, client, searchParams, state);

        const response = await authorizationCodeGrantRequest(
          as,
          client,
          oidcClientAuth,
          params,
          redirectURI,
          code_verifier,
          {
            [allowInsecureRequests]: shouldAllowHttpRequest(),
            additionalParameters: [['client_id', client.client_id]]
          }
        );

        const result = await processAuthorizationCodeResponse(as, client, response, {
          requireIdToken: true,
          expectedNonce: nonce
        });

        if (!result.id_token) {
          notification.error({
            message: 'OIDC: Proccess Authorization Code Grant Response error',
            placement: 'bottomRight'
          });
          return;
        }

        onLogin(result.id_token, result.refresh_token);

        // Always leave the callback URL. Staying on /login?code=... renders
        // the login button while a token is already stored, and a reload
        // replays the authorization code, which the provider refuses and
        // may answer by revoking the session it just issued (RFC 6749
        // 4.1.2). Logout lands on a bare /login, so the next sign-in is
        // exactly the case with no redirectTo: fall back to home.
        const redirectTo = platformRedirect
          ? new URLSearchParams(platformRedirect).get(redirectToQueryParam)
          : null;

        window.location.replace(
          window.location.origin + (isSafeRedirectPath(redirectTo) ? redirectTo : paths.home)
        );
      } catch (err) {
        if (err instanceof AuthorizationResponseError) {
          notification.error({
            message: 'OIDC: Validation Auth Response error',
            placement: 'bottomRight'
          });
          return;
        }

        if (err instanceof WWWAuthenticateChallengeError) {
          notification.error({
            message: 'OIDC: Parsing Authenticate Challenges error',
            placement: 'bottomRight'
          });
          return;
        }

        notification.error({
          message: `OIDC: ${oidcErrorMessage(err)}`,
          placement: 'bottomRight'
        });
      }
    })();
  }, [as, client, location]);

  return (
    <Button onClick={login} block size='large' loading={isFetching} disabled={!issuerUrl}>
      SSO Login
    </Button>
  );
};
