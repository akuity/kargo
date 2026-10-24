import { AuthorizationServer, ClientAuth } from 'oauth4webapi';

import { OIDCConfig } from '@ui/gen/api/v2/models';

export const oidcSessionStorageKeys = {
  codeVerifier: 'PKCE_code_verifier',
  state: 'PKCE_state',
  nonce: 'OIDC_nonce',
  platformRedirect: 'platform_redirect'
} as const;

const oidcCallbackQueryKeys = [
  'code',
  'state',
  'session_state',
  'iss',
  'error',
  'error_description',
  'error_uri'
] as const;

export const consumeOIDCCallbackArtifacts = (
  storage: Pick<Storage, 'removeItem'>,
  callbackURL: URL,
  replaceURL: (url: string) => void
) => {
  Object.values(oidcSessionStorageKeys).forEach((key) => storage.removeItem(key));
  oidcCallbackQueryKeys.forEach((key) => callbackURL.searchParams.delete(key));
  replaceURL(`${callbackURL.pathname}${callbackURL.search}${callbackURL.hash}`);
};

export const oidcErrorMessage = (err: unknown) => {
  if (!(err instanceof Error)) {
    return 'Unexpected error';
  }

  const details: string[] = [];
  if (typeof err.cause === 'object' && err.cause !== null && 'claim' in err.cause) {
    const claim = err.cause.claim;
    if (typeof claim === 'string') {
      details.push(`claim: ${claim}`);
    }
  }

  if ('code' in err && typeof err.code === 'string') {
    details.push(`code: ${err.code}`);
  }

  const message = err.message || err.name || 'Unexpected error';
  return details.length ? `${message} (${details.join(', ')})` : message;
};

export const oidcClientAuth: ClientAuth = () => {
  // equivalent function for token_endpoint_auth_method: 'none'
};

export const shouldAllowIdpHttpRequest = () => true;

export const getOIDCScopes = (userOIDCConfig: OIDCConfig, idp: AuthorizationServer) => {
  const scopes = [...(userOIDCConfig.scopes ?? [])];

  // add offline_access scope automatically only if it is supported by IDP
  if (!scopes.includes('offline_access') && idp.scopes_supported?.includes('offline_access')) {
    scopes.push('offline_access');
  }

  return scopes;
};
