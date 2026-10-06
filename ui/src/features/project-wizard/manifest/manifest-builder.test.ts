import { expect, test } from 'vitest';
import yaml from 'yaml';

import {
  exampleStages,
  exampleWarehouse,
  initialBasicsState,
  initialCredential,
  initialWizardState,
  isValidProjectName,
  normalizeCredential
} from '../types';

import {
  MASKED_SECRET,
  basicsFromYaml,
  creationManifests,
  credentialSecretManifest,
  credentialsFromYaml,
  orderedManifests,
  projectConfigManifest,
  projectManifest,
  resourceKey,
  resourceList,
  stageManifest,
  warehouseManifest,
  warehousesFromYaml,
  yamlForStep
} from './manifest-builder';

test('isValidProjectName enforces RFC 1123 labels', () => {
  expect(isValidProjectName('my-app-delivery')).toBe(true);
  expect(isValidProjectName('a')).toBe(true);
  expect(isValidProjectName('')).toBe(false);
  expect(isValidProjectName('-leading-dash')).toBe(false);
  expect(isValidProjectName('trailing-dash-')).toBe(false);
  expect(isValidProjectName('UpperCase')).toBe(false);
  expect(isValidProjectName('dot.ted')).toBe(false);
  expect(isValidProjectName('a'.repeat(64))).toBe(false);
  expect(isValidProjectName('a'.repeat(63))).toBe(true);
});

test('projectManifest with only a name omits annotations', () => {
  const manifest = projectManifest({ ...initialBasicsState(), name: 'my-project' });
  expect(manifest).toEqual({
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Project',
    metadata: { name: 'my-project' }
  });
});

test('projectManifest maps description to the description annotation', () => {
  const manifest = projectManifest({
    ...initialBasicsState(),
    name: 'my-project',
    description: 'A test project'
  });
  expect(manifest.metadata.annotations).toEqual({
    'kargo.akuity.io/description': 'A test project'
  });
});

test('resourceList is empty until the project name is valid', () => {
  const state = initialWizardState();
  expect(resourceList(state)).toEqual([]);
  state.basics.name = 'Bad Name';
  expect(resourceList(state)).toEqual([]);
  state.basics.name = 'good-name';
  expect(resourceList(state)).toEqual([{ kind: 'Project', name: 'good-name' }]);
});

// Progress is overlaid on the resource list through Maps keyed on resourceKey,
// in the review step and in the retry merge. A Map silently keeps only the last
// entry per key, so duplicates in a draft must still key distinctly — otherwise
// the resource that was created reports the duplicate's failure.
test('resourceList keys stay unique when two resources share a name', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.stages = [
    { name: 'dev', requestedFreight: [], steps: [] },
    { name: 'dev', requestedFreight: [], steps: [] }
  ];
  const keys = resourceList(state).map(resourceKey);
  expect(keys).toEqual(['Project/my-project', 'Stage/dev', 'Stage/dev#1']);
  expect(new Set(keys).size).toBe(keys.length);
  // The engine keys off the same refs, so they carry the ordinal too.
  expect(creationManifests(state).map(resourceKey)).toEqual(keys);
});

// The ordinal counts occurrences of a kind/name, not list position: a retry
// matches a freshly generated list against the previous run's, and a resource
// must keep its key when an unrelated one appears ahead of it.
test('resourceKey survives an unrelated resource being inserted ahead of it', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.stages = [{ name: 'dev', requestedFreight: [], steps: [] }];
  expect(resourceList(state).map(resourceKey)).toEqual(['Project/my-project', 'Stage/dev']);

  state.warehouses = [{ name: 'wh', spec: { subscriptions: [] } }];
  expect(resourceList(state).map(resourceKey)).toEqual([
    'Project/my-project',
    'Warehouse/wh',
    // Moved from index 1 to index 2; the key is unchanged.
    'Stage/dev'
  ]);
});

test('yamlForStep renders the project manifest for basics and review', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  for (const step of ['basics', 'review'] as const) {
    const parsed = yaml.parse(yamlForStep(state, step));
    expect(parsed.kind).toBe('Project');
    expect(parsed.metadata.name).toBe('my-project');
  }
});

test('basicsFromYaml round-trips projectManifest output', () => {
  const basics = { ...initialBasicsState(), name: 'my-project', description: 'A test project' };
  const roundTripped = basicsFromYaml(yaml.stringify(projectManifest(basics)), basics);
  expect(roundTripped).toEqual(basics);
});

test('basicsFromYaml applies edits from the YAML', () => {
  const prev = { ...initialBasicsState(), name: 'old-name' };
  const next = basicsFromYaml(
    [
      'apiVersion: kargo.akuity.io/v1alpha1',
      'kind: Project',
      'metadata:',
      '  name: new-name',
      '  annotations:',
      '    kargo.akuity.io/description: hello'
    ].join('\n'),
    prev
  );
  expect(next.name).toBe('new-name');
  expect(next.description).toBe('hello');
});

test('basicsFromYaml rejects unusable input', () => {
  const prev = initialBasicsState();
  expect(() => basicsFromYaml('# just a comment', prev)).toThrow('Expected a YAML mapping');
  expect(() => basicsFromYaml('kind: Stage\nmetadata:\n  name: x', prev)).toThrow(
    'Expected kind: Project'
  );
  expect(() => basicsFromYaml('{invalid', prev)).toThrow();
  // A top-level sequence is a mapping error, not a kind error.
  expect(() => basicsFromYaml('- kind: Project', prev)).toThrow('Expected a YAML mapping');
});

test('basicsFromYaml degrades unusable fields instead of failing the document', () => {
  const prev = { ...initialBasicsState(), name: 'old-name', description: 'old' };

  // Absent metadata, a non-string name, and annotations that are not a mapping
  // all fall back rather than throwing -- the rail commits while the user is
  // still typing, so a half-written document must map to a partial draft.
  expect(basicsFromYaml('kind: Project', prev)).toEqual({ ...prev, name: '', description: '' });
  expect(basicsFromYaml('kind: Project\nmetadata:\n  name: 42', prev).name).toBe('');
  expect(
    basicsFromYaml('kind: Project\nmetadata:\n  name: x\n  annotations: nope', prev).description
  ).toBe('');

  // Annotation values are coerced to strings, as Kubernetes requires.
  expect(
    basicsFromYaml(
      [
        'kind: Project',
        'metadata:',
        '  name: x',
        '  annotations:',
        '    kargo.akuity.io/description: 7'
      ].join('\n'),
      prev
    ).description
  ).toBe('7');
});

test('a parse failure throws only its message, not a serialized error', () => {
  // The rail shows the thrown message verbatim, so it has to stay readable --
  // a raw ZodError stringifies to a JSON dump of its issues.
  expect(() => basicsFromYaml('kind: Stage', initialBasicsState())).toThrow(
    /^Expected kind: Project$/
  );
});

test('credentialSecretManifest builds a labeled Secret per auth method', () => {
  const userpass = {
    ...initialCredential('git'),
    name: 'github-creds',
    description: 'org creds',
    repoURL: 'https://github.com/acme/.*',
    repoURLIsRegex: true,
    username: 'bot',
    password: 'secret'
  };
  expect(credentialSecretManifest(userpass, 'my-project')).toEqual({
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: 'github-creds',
      namespace: 'my-project',
      labels: { 'kargo.akuity.io/cred-type': 'git' },
      annotations: { 'kargo.akuity.io/description': 'org creds' }
    },
    stringData: {
      repoURL: 'https://github.com/acme/.*',
      repoURLIsRegex: 'true',
      username: 'bot',
      password: 'secret'
    }
  });

  const gcp = {
    ...initialCredential('image'),
    name: 'gar-creds',
    auth: 'gcp-ar' as const,
    repoURL: 'us-docker.pkg.dev/acme/repo',
    gcpServiceAccountKey: '{"type":"service_account"}'
  };
  expect(credentialSecretManifest(gcp, 'my-project')?.stringData?.gcpServiceAccountKey).toBe(
    btoa('{"type":"service_account"}')
  );

  const ambient = { ...initialCredential('image'), name: 'ecr', auth: 'ambient' as const };
  expect(credentialSecretManifest(ambient, 'my-project')).toBeNull();
});

test('orderedManifests places credential Secrets after the Project', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.credentials = [
    { ...initialCredential('git'), name: 'git-creds', repoURL: 'https://example.com/repo' },
    { ...initialCredential('image'), name: 'ambient-creds', auth: 'ambient' }
  ];
  expect(resourceList(state)).toEqual([
    { kind: 'Project', name: 'my-project' },
    { kind: 'Secret', name: 'git-creds' }
  ]);
  expect(orderedManifests(state)[1].metadata.namespace).toBe('my-project');
});

test('credentialsFromYaml round-trips and preserves ambient credentials', () => {
  const prev = [
    {
      ...initialCredential('git'),
      name: 'gh-app',
      auth: 'github-app' as const,
      repoURL: 'https://github.com/acme/repo',
      githubAppClientID: 'Iv1.abc',
      githubAppInstallationID: '2',
      githubAppPrivateKey: 'PEM'
    },
    {
      ...initialCredential('image'),
      name: 'gar-creds',
      auth: 'gcp-ar' as const,
      repoURL: 'us-docker.pkg.dev/acme/repo',
      gcpServiceAccountKey: '{"k":"v"}'
    },
    { ...initialCredential('image'), name: 'ecr-ambient', auth: 'ambient' as const }
  ];
  const text = prev
    .map((c) => credentialSecretManifest(c, 'my-project'))
    .filter(Boolean)
    .map((m) => yaml.stringify(m))
    .join('---\n');
  expect(credentialsFromYaml(text, prev)).toEqual([
    prev[0],
    prev[1],
    prev[2] // ambient carried over from previous state, appended last
  ]);
});

test('normalizeCredential backfills new fields and coerces stale types', () => {
  // A draft from when the wizard also had generic secrets, missing newer
  // fields like githubAppClientID.
  const migrated = normalizeCredential({
    name: 'token',
    type: 'generic',
    repoURL: 'https://example.com/repo'
  });
  // generic is no longer a repo credential type; coerce back to git
  expect(migrated.type).toBe('git');
  expect(migrated.auth).toBe('userpass');
  expect(migrated.githubAppClientID).toBe('');
  // and a normalized credential builds a manifest without throwing
  expect(credentialSecretManifest(migrated, 'my-project')?.metadata.labels).toEqual({
    'kargo.akuity.io/cred-type': 'git'
  });
});

test('credentialsFromYaml rejects unusable input', () => {
  expect(() => credentialsFromYaml('kind: Stage\nmetadata: {name: x}', [])).toThrow(
    'Expected kind: Secret'
  );
  expect(() => credentialsFromYaml('kind: Secret\nmetadata: {name: x}', [])).toThrow(/needs label/);
  // generic is no longer accepted by the credentials step
  expect(() =>
    credentialsFromYaml(
      'kind: Secret\nmetadata:\n  name: x\n  labels:\n    kargo.akuity.io/cred-type: generic',
      []
    )
  ).toThrow(/needs label/);
});

test('yamlForStep for credentials renders multi-doc with ambient note', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.credentials = [
    { ...initialCredential('git'), name: 'git-creds', repoURL: 'https://example.com/repo' },
    { ...initialCredential('image'), name: 'ecr-ambient', auth: 'ambient' }
  ];
  const text = yamlForStep(state, 'credentials');
  expect(text).toContain('kind: Secret');
  expect(text).toContain('ecr-ambient: ambient');
  const docs = yaml.parseAllDocuments(text);
  expect(docs).toHaveLength(2); // Secret + trailing comment doc
});

test('yamlForStep masks credential secrets in the preview but not in creation', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.credentials = [
    {
      ...initialCredential('git'),
      name: 'gh',
      repoURL: 'https://github.com/acme/repo',
      username: 'bot',
      password: 'super-secret'
    }
  ];
  const preview = yamlForStep(state, 'credentials');
  expect(preview).not.toContain('super-secret');
  expect(preview).toContain(MASKED_SECRET);
  // non-secret data stays visible
  expect(preview).toContain('bot');
  expect(preview).toContain('https://github.com/acme/repo');
  // the creation manifest still carries the real secret
  expect(creationManifests(state)[1].yaml).toContain('super-secret');
});

test('credentialsFromYaml preserves a masked secret from the previous state', () => {
  const prev = [
    {
      ...initialCredential('git'),
      name: 'gh',
      repoURL: 'https://github.com/acme/repo',
      username: 'bot',
      password: 'real-secret'
    }
  ];
  const state = {
    ...initialWizardState(),
    basics: { name: 'my-project', description: '' },
    credentials: prev
  };
  // The masked preview the user sees, with a non-secret field edited and the
  // masked password left in place.
  const edited = yamlForStep(state, 'credentials').replace(
    'https://github.com/acme/repo',
    'https://github.com/acme/other'
  );
  const result = credentialsFromYaml(edited, prev);
  expect(result[0].repoURL).toBe('https://github.com/acme/other');
  // the masked placeholder is restored to the real secret, never persisted as-is
  expect(result[0].password).toBe('real-secret');
});

test('warehouseManifest wraps the form spec and cleans empties', () => {
  const draft = {
    name: 'my-warehouse',
    spec: {
      interval: '5m0s',
      subscriptions: [{ image: { repoURL: 'ghcr.io/acme/app' } }],
      emptyBlock: {}
    }
  };
  const manifest = warehouseManifest(draft, 'my-project');
  expect(manifest).toEqual({
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Warehouse',
    metadata: { name: 'my-warehouse', namespace: 'my-project' },
    spec: {
      interval: '5m0s',
      subscriptions: [{ image: { repoURL: 'ghcr.io/acme/app' } }]
    }
  });
  // the source draft is not mutated by cleanEmptyObjectValues (it works on a clone)
  expect(draft.spec.emptyBlock).toEqual({});
});

test('orderedManifests appends warehouses after credentials', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.credentials = [
    { ...initialCredential('git'), name: 'git-creds', repoURL: 'https://example.com/repo' }
  ];
  state.warehouses = [{ name: 'wh', spec: { subscriptions: [] } }];
  expect(resourceList(state)).toEqual([
    { kind: 'Project', name: 'my-project' },
    { kind: 'Secret', name: 'git-creds' },
    { kind: 'Warehouse', name: 'wh' }
  ]);
});

test('warehousesFromYaml round-trips warehouseManifest output', () => {
  const drafts = [
    { name: 'wh-a', spec: { interval: '5m0s', subscriptions: [{ git: { repoURL: 'x' } }] } },
    { name: 'wh-b', spec: { interval: '10m0s', subscriptions: [{ image: { repoURL: 'y' } }] } }
  ];
  const text = drafts.map((w) => yaml.stringify(warehouseManifest(w, 'my-project'))).join('---\n');
  expect(warehousesFromYaml(text)).toEqual(drafts);
});

test('exampleWarehouse builds the akuity kargo-simple Warehouse manifest', () => {
  const manifest = warehouseManifest(exampleWarehouse(), 'my-project');
  expect(manifest).toEqual({
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Warehouse',
    metadata: { name: 'guestbook', namespace: 'my-project' },
    spec: {
      subscriptions: [
        {
          image: {
            repoURL: 'ghcr.io/akuity/guestbook',
            imageSelectionStrategy: 'SemVer'
          }
        }
      ]
    }
  });
});

test('warehousesFromYaml rejects non-Warehouse input', () => {
  expect(() => warehousesFromYaml('kind: Secret\nmetadata: {name: x}')).toThrow(
    'Expected kind: Warehouse'
  );
});

test('warehousesFromYaml falls back to an empty spec when the spec is not a mapping', () => {
  const empty = { subscriptions: [] };
  expect(warehousesFromYaml('kind: Warehouse\nmetadata: {name: wh}')).toEqual([
    { name: 'wh', spec: empty }
  ]);
  expect(warehousesFromYaml('kind: Warehouse\nmetadata: {name: wh}\nspec: nope')).toEqual([
    { name: 'wh', spec: empty }
  ]);
  // A sequence is not a valid spec either. Kubernetes would reject it, so it
  // degrades here rather than reaching creation.
  expect(warehousesFromYaml('kind: Warehouse\nmetadata: {name: wh}\nspec:\n- a')).toEqual([
    { name: 'wh', spec: empty }
  ]);
});

test('yamlForStep renders warehouses or a placeholder', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  expect(yamlForStep(state, 'warehouses').startsWith('#')).toBe(true);
  state.warehouses = [{ name: 'wh', spec: { subscriptions: [] } }];
  const text = yamlForStep(state, 'warehouses');
  expect(text).toContain('kind: Warehouse');
  expect(text).toContain('name: wh');
});

test('stageManifest maps color, requested freight, and promotion steps', () => {
  const manifest = stageManifest(
    {
      name: 'dev',
      color: 'red',
      requestedFreight: [
        { origin: { kind: 'Warehouse', name: 'guestbook' }, sources: { direct: true, stages: [] } }
      ],
      steps: [
        // shape of a reused PromotionStepsWizard RunnerWithConfiguration
        {
          identifier: 'git-clone',
          config: {},
          state: { repoURL: 'x' },
          as: 'clone',
          continueOnError: false
        }
      ]
    },
    'my-project'
  );
  expect(manifest).toEqual({
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Stage',
    metadata: {
      name: 'dev',
      namespace: 'my-project',
      annotations: { 'kargo.akuity.io/color': 'red' }
    },
    spec: {
      requestedFreight: [
        { origin: { kind: 'Warehouse', name: 'guestbook' }, sources: { direct: true, stages: [] } }
      ],
      promotionTemplate: {
        spec: {
          steps: [
            { uses: 'git-clone', as: 'clone', continueOnError: false, config: { repoURL: 'x' } }
          ]
        }
      }
    }
  });
});

test('stageManifest omits promotionTemplate when there are no steps', () => {
  const manifest = stageManifest(
    {
      name: 'dev',
      requestedFreight: [
        { origin: { kind: 'Warehouse', name: 'guestbook' }, sources: { direct: true } }
      ],
      steps: []
    },
    'my-project'
  );
  expect(manifest.metadata.annotations).toBeUndefined();
  expect(manifest.spec?.promotionTemplate).toBeUndefined();
  expect(manifest.spec?.requestedFreight).toBeDefined();
});

test('orderedManifests appends stages after warehouses', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.warehouses = [{ name: 'guestbook', spec: { subscriptions: [] } }];
  state.stages = exampleStages('guestbook');
  expect(resourceList(state)).toEqual([
    { kind: 'Project', name: 'my-project' },
    { kind: 'Warehouse', name: 'guestbook' },
    { kind: 'Stage', name: 'dev' },
    { kind: 'Stage', name: 'staging' },
    { kind: 'Stage', name: 'prod' }
  ]);
});

test('exampleStages chains dev -> staging -> prod from the guestbook warehouse', () => {
  const stages = exampleStages('guestbook');
  expect(stages.map((s) => s.name)).toEqual(['dev', 'staging', 'prod']);
  expect(stages[0].requestedFreight[0].sources?.direct).toBe(true);
  expect(stages[1].requestedFreight[0].sources?.stages).toEqual(['dev']);
  expect(stages[2].requestedFreight[0].sources?.stages).toEqual(['staging']);
  expect(stages.every((s) => s.requestedFreight[0].origin.name === 'guestbook')).toBe(true);
});

test('projectConfigManifest maps each selector type to a stageSelector', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.policies = [
    { selectorType: 'exact', value: 'dev', matchLabels: {}, autoPromotionEnabled: true },
    { selectorType: 'regex', value: '^prod-.*$', matchLabels: {}, autoPromotionEnabled: false },
    { selectorType: 'glob', value: 'stg-*', matchLabels: {}, autoPromotionEnabled: true },
    {
      selectorType: 'labels',
      value: '',
      matchLabels: { env: 'prod' },
      autoPromotionEnabled: true
    }
  ];
  expect(projectConfigManifest(state)).toEqual({
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'ProjectConfig',
    metadata: { name: 'my-project', namespace: 'my-project' },
    spec: {
      promotionPolicies: [
        { autoPromotionEnabled: true, stageSelector: { name: 'dev' } },
        { autoPromotionEnabled: false, stageSelector: { name: 'regex:^prod-.*$' } },
        { autoPromotionEnabled: true, stageSelector: { name: 'glob:stg-*' } },
        { autoPromotionEnabled: true, stageSelector: { matchLabels: { env: 'prod' } } }
      ]
    }
  });
});

test('projectConfigManifest is null and omitted from ordering when no policies', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  expect(projectConfigManifest(state)).toBeNull();
  expect(resourceList(state)).toEqual([{ kind: 'Project', name: 'my-project' }]);
});

test('orderedManifests appends ProjectConfig last', () => {
  const state = initialWizardState();
  state.basics.name = 'my-project';
  state.stages = [{ name: 'dev', requestedFreight: [], steps: [] }];
  state.policies = [
    { selectorType: 'exact', value: 'dev', matchLabels: {}, autoPromotionEnabled: true }
  ];
  expect(resourceList(state)).toEqual([
    { kind: 'Project', name: 'my-project' },
    { kind: 'Stage', name: 'dev' },
    { kind: 'ProjectConfig', name: 'my-project' }
  ]);
});

test('yamlForStep falls back to comment placeholders', () => {
  const state = initialWizardState();
  // No name yet -- both implemented steps show a placeholder.
  expect(yamlForStep(state, 'basics').startsWith('#')).toBe(true);
  expect(yamlForStep(state, 'review').startsWith('#')).toBe(true);
  // Unimplemented steps always show a placeholder.
  state.basics.name = 'my-project';
  expect(yamlForStep(state, 'warehouses').startsWith('#')).toBe(true);
});
