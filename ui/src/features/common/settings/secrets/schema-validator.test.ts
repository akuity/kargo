import { expect, test } from 'vitest';

import { createFormSchema, repoUrlError } from './schema-validator';

test('repoUrlError accepts URLs the credential type can use', () => {
  expect(repoUrlError('git', 'https://github.com/akuity/kargo.git')).toBeUndefined();
  expect(repoUrlError('git', 'ssh://git@github.com/akuity/kargo.git')).toBeUndefined();
  expect(repoUrlError('helm', 'https://charts.example.com')).toBeUndefined();
  expect(repoUrlError('helm', 'oci://ghcr.io/akuity/charts')).toBeUndefined();
  expect(repoUrlError('image', 'ghcr.io/akuity/kargo')).toBeUndefined();
});

test('repoUrlError rejects URLs the credential type cannot use', () => {
  expect(repoUrlError('git', 'not a url')).toBe('Repo URL must be a valid git URL.');
  // Helm charts are served over HTTP(S) or OCI -- a git transport is neither.
  expect(repoUrlError('helm', 'ssh://git@github.com/akuity/charts.git')).toBe(
    'Repo URL must be a valid Helm chart repository.'
  );
  // An image reference is a registry path, not a URL with a scheme.
  expect(repoUrlError('image', 'https://ghcr.io/akuity/kargo')).toBe(
    'Repo URL must be a valid container registry.'
  );
});

test('repoUrlError leaves unchecked input alone', () => {
  // A pattern matches repo URLs rather than being one.
  expect(repoUrlError('git', '^https://github.com/akuity/.*$', true)).toBeUndefined();
  // Emptiness is the "required" checks' business, and generic secrets have no
  // repo URL rules at all.
  expect(repoUrlError('git', '')).toBeUndefined();
  expect(repoUrlError('git', undefined)).toBeUndefined();
  expect(repoUrlError('generic', 'not a url')).toBeUndefined();
});

test('createFormSchema reports the type-specific repo URL error on repoUrl', () => {
  const result = createFormSchema(false).safeParse({
    name: 'my-creds',
    type: 'helm',
    repoUrl: 'ssh://git@github.com/akuity/charts.git',
    username: 'someone',
    password: 'secret'
  });
  expect(result.success).toBe(false);
  expect(result.error?.issues).toContainEqual(
    expect.objectContaining({
      path: ['repoUrl'],
      message: 'Repo URL must be a valid Helm chart repository.'
    })
  );
});

// The form validates as the user types, so a bad URL has to be reported even
// while other fields are still empty.
test('createFormSchema reports the repo URL error alongside other field errors', () => {
  const result = createFormSchema(false).safeParse({
    name: '',
    type: 'image',
    repoUrl: 'https://ghcr.io/akuity/kargo'
  });
  expect(result.success).toBe(false);
  expect(result.error?.issues.map((issue) => issue.path.join('.'))).toEqual(
    expect.arrayContaining(['name', 'username', 'password', 'repoUrl'])
  );
});

test('createFormSchema skips repo URL rules for generic secrets', () => {
  const result = createFormSchema(true).safeParse({
    name: 'my-secret',
    type: 'generic',
    repoUrl: 'not a url'
  });
  expect(result.success).toBe(true);
});
