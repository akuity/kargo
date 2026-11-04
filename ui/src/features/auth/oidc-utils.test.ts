import { describe, expect, test } from 'vitest';

import {
  consumeOIDCCallbackArtifacts,
  oidcErrorMessage,
  oidcSessionStorageKeys
} from './oidc-utils';

describe('consumeOIDCCallbackArtifacts', () => {
  test('removes one-time browser state and callback parameters', () => {
    const removedKeys: string[] = [];
    let replacementURL = '';
    const callbackURL = new URL(
      'https://kargo.example.com/login?code=secret&state=nonce&session_state=session&iss=issuer&redirectTo=%2Fprojects#tab'
    );

    consumeOIDCCallbackArtifacts(
      { removeItem: (key) => removedKeys.push(key) },
      callbackURL,
      (url) => {
        replacementURL = url;
      }
    );

    expect(removedKeys).toEqual(Object.values(oidcSessionStorageKeys));
    expect(replacementURL).toBe('/login?redirectTo=%2Fprojects#tab');
  });

  test('removes OAuth error response parameters', () => {
    let replacementURL = '';
    const callbackURL = new URL(
      'https://kargo.example.com/login?error=access_denied&error_description=Denied&error_uri=https%3A%2F%2Fidp.example.com%2Ferrors&state=nonce'
    );

    consumeOIDCCallbackArtifacts({ removeItem: () => undefined }, callbackURL, (url) => {
      replacementURL = url;
    });

    expect(replacementURL).toBe('/login');
  });
});

describe('oidcErrorMessage', () => {
  test('shows safe comparison details without serializing claim values', () => {
    const err = Object.assign(
      new Error('unexpected JWT claim value', {
        cause: {
          claim: 'aud',
          expected: 'client-id',
          claims: { email: 'person@example.com' }
        }
      }),
      { code: 'OAUTH_JWT_CLAIM_COMPARISON_FAILED' }
    );

    const message = oidcErrorMessage(err);

    expect(message).toBe(
      'unexpected JWT claim value (claim: aud, code: OAUTH_JWT_CLAIM_COMPARISON_FAILED)'
    );
    expect(message).not.toContain('client-id');
    expect(message).not.toContain('person@example.com');
  });

  test('handles ordinary and unknown errors', () => {
    expect(oidcErrorMessage(new Error('network failed'))).toBe('network failed');
    expect(oidcErrorMessage({ message: 'not an Error' })).toBe('Unexpected error');
  });
});
