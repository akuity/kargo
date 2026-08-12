import { expect, test } from 'vitest';

import { CredentialData, initialCredential } from '../types';

import { isValidCredential, repoUrlError } from './credential-validation';

const credential = (overrides: Partial<CredentialData>): CredentialData => ({
  ...initialCredential(overrides.type),
  ...overrides
});

// The wizard's field names differ from the settings form's (repoURL vs
// repoUrl), so the mapping onto the shared rule is what can break here.
test('repoUrlError applies the shared repo URL rules to a credential', () => {
  expect(
    repoUrlError(credential({ type: 'helm', repoURL: 'oci://ghcr.io/akuity' }))
  ).toBeUndefined();
  expect(repoUrlError(credential({ type: 'git', repoURL: 'not a url' }))).toBe(
    'Repo URL must be a valid git URL.'
  );
  expect(
    repoUrlError(credential({ type: 'git', repoURL: 'github.com/*', repoURLIsRegex: true }))
  ).toBeUndefined();
});

test('isValidCredential needs a DNS name, a usable repo URL, and complete auth', () => {
  const valid = credential({
    name: 'github-creds',
    type: 'git',
    repoURL: 'https://github.com/akuity/kargo.git',
    auth: 'userpass',
    username: 'someone',
    password: 'secret'
  });
  expect(isValidCredential(valid)).toBe(true);
  expect(isValidCredential({ ...valid, name: 'Not_A_DNS_Name' })).toBe(false);
  expect(isValidCredential({ ...valid, repoURL: '' })).toBe(false);
  expect(isValidCredential({ ...valid, repoURL: 'not a url' })).toBe(false);
  expect(isValidCredential({ ...valid, password: '' })).toBe(false);
});

test('isValidCredential checks the fields each auth method actually needs', () => {
  const base = credential({
    name: 'image-creds',
    type: 'image',
    repoURL: 'ghcr.io/akuity/kargo'
  });
  expect(isValidCredential({ ...base, auth: 'aws-ecr' })).toBe(false);
  expect(
    isValidCredential({
      ...base,
      auth: 'aws-ecr',
      awsRegion: 'us-east-1',
      awsAccessKeyID: 'AKIA',
      awsSecretAccessKey: 'secret'
    })
  ).toBe(true);
  // Ambient credentials store nothing, so there is nothing more to fill in.
  expect(isValidCredential({ ...base, auth: 'ambient' })).toBe(true);
});
