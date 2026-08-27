import { expect, test } from 'vitest';

import {
  credentialSecretFields,
  initialCredential,
  initialWizardState,
  normalizeBasics,
  normalizePolicy,
  normalizeStage,
  normalizeWarehouse,
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

// The normalize* functions read drafts restored from localStorage, so their
// input is whatever an older version of the wizard wrote -- or anything at all.

test('normalizeBasics coerces a persisted draft', () => {
  expect(normalizeBasics({ name: 'p', description: 'd' })).toEqual({
    name: 'p',
    description: 'd'
  });

  const empty = { name: '', description: '' };
  expect(normalizeBasics({})).toEqual(empty);
  expect(normalizeBasics(undefined)).toEqual(empty);
  expect(normalizeBasics(null)).toEqual(empty);
  expect(normalizeBasics('nope')).toEqual(empty);
  // hydrate used to spread this slice in unchecked, so a non-string name
  // reached BasicsState.name and, from there, metadata.name in the manifest.
  expect(normalizeBasics({ name: 42, description: [] })).toEqual(empty);
});

test('normalizeWarehouse coerces a persisted draft', () => {
  const empty = { name: '', spec: { subscriptions: [] } };
  expect(normalizeWarehouse({ name: 'wh', spec: { interval: '5m' } })).toEqual({
    name: 'wh',
    spec: { interval: '5m' }
  });
  // Absent, wrong-typed, and non-object input all fall back.
  expect(normalizeWarehouse({})).toEqual(empty);
  expect(normalizeWarehouse(undefined)).toEqual(empty);
  expect(normalizeWarehouse(null)).toEqual(empty);
  expect(normalizeWarehouse('nope')).toEqual(empty);
  expect(normalizeWarehouse({ name: 42, spec: 'nope' })).toEqual(empty);
  // A sequence is not a valid spec. The hand-written check let this through,
  // since typeof [] === 'object'.
  expect(normalizeWarehouse({ name: 'wh', spec: [] })).toEqual({ ...empty, name: 'wh' });
});

test('normalizeStage coerces a persisted draft', () => {
  const freight = [{ origin: { kind: 'Warehouse', name: 'wh' } }];
  expect(
    normalizeStage({ name: 'dev', color: 'red', requestedFreight: freight, steps: [] })
  ).toEqual({ name: 'dev', color: 'red', requestedFreight: freight, steps: [] });

  const empty = { name: '', color: undefined, requestedFreight: [], steps: [] };
  expect(normalizeStage({})).toEqual(empty);
  expect(normalizeStage(undefined)).toEqual(empty);
  expect(normalizeStage('nope')).toEqual(empty);
  // A non-string color is dropped rather than kept, and non-arrays empty out.
  expect(normalizeStage({ name: 'dev', color: 7, requestedFreight: 'x', steps: {} })).toEqual({
    ...empty,
    name: 'dev'
  });
});

test('normalizePolicy coerces a persisted draft', () => {
  expect(
    normalizePolicy({
      selectorType: 'regex',
      value: '^prod-.*$',
      matchLabels: { env: 'prod' },
      autoPromotionEnabled: true
    })
  ).toEqual({
    selectorType: 'regex',
    value: '^prod-.*$',
    matchLabels: { env: 'prod' },
    autoPromotionEnabled: true
  });

  const empty = {
    selectorType: 'exact',
    value: '',
    matchLabels: {},
    autoPromotionEnabled: false
  };
  expect(normalizePolicy({})).toEqual(empty);
  expect(normalizePolicy(undefined)).toEqual(empty);
  expect(normalizePolicy('nope')).toEqual(empty);

  // Every valid selector kind survives; anything else falls back to 'exact'.
  for (const selectorType of ['exact', 'regex', 'glob', 'labels'] as const) {
    expect(normalizePolicy({ selectorType }).selectorType).toBe(selectorType);
  }
  expect(normalizePolicy({ selectorType: 'bogus' }).selectorType).toBe('exact');

  // autoPromotionEnabled keeps the truthiness coercion the old `!!` applied.
  expect(normalizePolicy({ autoPromotionEnabled: 'yes' }).autoPromotionEnabled).toBe(true);
  expect(normalizePolicy({ autoPromotionEnabled: 0 }).autoPromotionEnabled).toBe(false);

  // matchLabels is Record<string, string>, so values are coerced -- the
  // hand-written version passed a persisted number through untouched.
  expect(normalizePolicy({ matchLabels: { env: 2 } }).matchLabels).toEqual({ env: '2' });
  expect(normalizePolicy({ matchLabels: [] }).matchLabels).toEqual({});
});

test('stripCredentialSecrets leaves other slices untouched', () => {
  const state = initialWizardState();
  state.basics = { name: 'my-project', description: 'desc' };
  const stripped = stripCredentialSecrets(state);
  expect(stripped.basics).toEqual({ name: 'my-project', description: 'desc' });
  expect(stripped.credentials).toEqual([]);
});
