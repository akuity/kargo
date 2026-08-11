import { expect, test } from 'vitest';

import {
  credentialSecretFields,
  initialCredential,
  initialWizardState,
  stripCredentialSecrets
} from './types';

test('stripCredentialSecrets blanks secret material but keeps identifiers', () => {
  const state = initialWizardState();
  state.credentials = [
    {
      ...initialCredential('git'),
      name: 'gh',
      repoURL: 'https://github.com/acme/repo',
      username: 'bot',
      password: 'p@ss',
      sshPrivateKey: 'SSH-PEM',
      githubAppClientID: 'Iv1.abc',
      githubAppPrivateKey: 'APP-PEM',
      awsAccessKeyID: 'AKIA',
      awsSecretAccessKey: 'shhh',
      gcpServiceAccountKey: '{"k":"v"}'
    }
  ];

  const cred = stripCredentialSecrets(state).credentials[0];

  // Every secret field is cleared.
  for (const field of credentialSecretFields) {
    expect(cred[field]).toBe('');
  }
  // Non-secret identifiers are preserved.
  expect(cred.name).toBe('gh');
  expect(cred.repoURL).toBe('https://github.com/acme/repo');
  expect(cred.username).toBe('bot');
  expect(cred.githubAppClientID).toBe('Iv1.abc');
  expect(cred.awsAccessKeyID).toBe('AKIA');

  // The input state is not mutated.
  expect(state.credentials[0].password).toBe('p@ss');
  expect(state.credentials[0].gcpServiceAccountKey).toBe('{"k":"v"}');
});

test('stripCredentialSecrets leaves other slices untouched', () => {
  const state = initialWizardState();
  state.basics = { name: 'my-project', description: 'desc' };
  const stripped = stripCredentialSecrets(state);
  expect(stripped.basics).toEqual({ name: 'my-project', description: 'desc' });
  expect(stripped.credentials).toEqual([]);
});
