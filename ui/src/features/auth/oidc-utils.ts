import { AuthorizationServer, ClientAuth } from 'oauth4webapi';

import { OIDCConfig } from '@ui/gen/service/v1alpha1/service_pb';

export const oidcClientAuth: ClientAuth = () => {
  // equivalent function for token_endpoint_auth_method: 'none'
};

export const shouldAllowIdpHttpRequest = () => true;

export const getOIDCScopes = (userOIDCConfig: OIDCConfig, idp: AuthorizationServer) => {
  const scopes = [...userOIDCConfig.scopes];

  // add offline_access scope automatically only if it is supported by IDP
  if (!scopes.includes('offline_access') && idp.scopes_supported?.includes('offline_access')) {
    scopes.push('offline_access');
  }

  return scopes;
};
