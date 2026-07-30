import yaml from 'yaml';

import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import { cleanEmptyObjectValues } from '@ui/utils/helpers';

import {
  BasicsState,
  CredentialData,
  CredentialType,
  PolicyDraft,
  StageDraft,
  StepId,
  WarehouseDraft,
  WizardState,
  initialCredential,
  isValidProjectName
} from '../types';

// Annotation/label keys mirror api/v1alpha1/{annotations,labels}.go
const annotationKeyDescription = DESCRIPTION_ANNOTATION_KEY;
const labelKeyCredentialType = 'kargo.akuity.io/cred-type';

export type ResourceRef = {
  kind: string;
  name: string;
};

type Manifest = {
  apiVersion: string;
  kind: string;
  metadata: {
    name: string;
    namespace?: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  type?: string;
  stringData?: Record<string, string>;
  spec?: Record<string, unknown>;
};

const base64Utf8 = (value: string) => btoa(String.fromCharCode(...new TextEncoder().encode(value)));

const tryDecodeBase64Utf8 = (value: string) => {
  try {
    return new TextDecoder().decode(Uint8Array.from(atob(value), (c) => c.charCodeAt(0)));
  } catch (_) {
    return value;
  }
};

const credTypes: readonly CredentialType[] = ['git', 'image', 'helm'];

export const projectManifest = (basics: BasicsState): Manifest => {
  const annotations: Record<string, string> = {};
  if (basics.description) {
    annotations[annotationKeyDescription] = basics.description;
  }
  return {
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Project',
    metadata: {
      name: basics.name,
      ...(Object.keys(annotations).length > 0 && { annotations })
    }
  };
};

const credentialStringData = (cred: CredentialData): Record<string, string> => {
  const data: Record<string, string> = { repoURL: cred.repoURL };
  if (cred.repoURLIsRegex) {
    data.repoURLIsRegex = 'true';
  }
  switch (cred.auth) {
    case 'userpass':
      data.username = cred.username;
      data.password = cred.password;
      break;
    case 'ssh':
      data.sshPrivateKey = cred.sshPrivateKey;
      break;
    case 'github-app':
      // Kargo prioritizes githubAppClientID (recommended by GitHub) over the
      // deprecated numeric githubAppID; emit whichever the user provided.
      if (cred.githubAppClientID) {
        data.githubAppClientID = cred.githubAppClientID;
      }
      if (cred.githubAppID) {
        data.githubAppID = cred.githubAppID;
      }
      data.githubAppInstallationID = cred.githubAppInstallationID;
      data.githubAppPrivateKey = cred.githubAppPrivateKey;
      break;
    case 'aws-ecr':
      data.awsRegion = cred.awsRegion;
      data.awsAccessKeyID = cred.awsAccessKeyID;
      data.awsSecretAccessKey = cred.awsSecretAccessKey;
      break;
    case 'gcp-ar':
      // pkg/credentials/gar expects the JSON key base64-encoded
      data.gcpServiceAccountKey = base64Utf8(cred.gcpServiceAccountKey);
      break;
  }
  return data;
};

// Returns null for ambient credentials: nothing is stored, the controller
// relies on IRSA / workload identity instead.
export const credentialSecretManifest = (
  cred: CredentialData,
  project: string
): Manifest | null => {
  if (cred.auth === 'ambient') {
    return null;
  }
  return {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: cred.name,
      namespace: project,
      labels: { [labelKeyCredentialType]: cred.type },
      ...(cred.description && {
        annotations: { [annotationKeyDescription]: cred.description }
      })
    },
    stringData: credentialStringData(cred)
  };
};

// Mirrors warehouseManifestsGen in features/utils/manifest-generator. Clones
// the spec so cleanEmptyObjectValues (which mutates) never touches wizard state.
export const warehouseManifest = (draft: WarehouseDraft, project: string): Manifest => ({
  apiVersion: 'kargo.akuity.io/v1alpha1',
  kind: 'Warehouse',
  metadata: { name: draft.name, namespace: project },
  spec: cleanEmptyObjectValues(structuredClone(draft.spec))
});

// Mirrors stageFormToYAML in features/stage/create-stage. Steps map from the
// reused PromotionStepsWizard's RunnerWithConfiguration to Stage promotion steps.
export const stageManifest = (draft: StageDraft, project: string): Manifest => {
  const steps = draft.steps.map((s) => ({
    uses: s.identifier,
    as: s.as || '',
    continueOnError: s.continueOnError || false,
    // Always an object: a null/omitted config fails backend validation with a
    // confusing root-type error rather than a useful "field required" one.
    config: s.state ?? {},
    vars: []
  }));
  return {
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'Stage',
    metadata: {
      name: draft.name,
      namespace: project,
      ...(draft.color && { annotations: { 'kargo.akuity.io/color': draft.color } })
    },
    spec: {
      requestedFreight: structuredClone(draft.requestedFreight),
      ...(steps.length > 0 && {
        promotionTemplate: { spec: cleanEmptyObjectValues(structuredClone({ steps })) }
      })
    }
  };
};

const promotionPolicySpec = (p: PolicyDraft): Record<string, unknown> => {
  const base = { autoPromotionEnabled: p.autoPromotionEnabled };
  switch (p.selectorType) {
    case 'regex':
      return { ...base, stageSelector: { name: `regex:${p.value}` } };
    case 'glob':
      return { ...base, stageSelector: { name: `glob:${p.value}` } };
    case 'labels':
      return { ...base, stageSelector: { matchLabels: { ...p.matchLabels } } };
    case 'exact':
    default:
      return { ...base, stageSelector: { name: p.value } };
  }
};

// A ProjectConfig holds project-scoped configuration: promotion policies (and,
// later, webhook receivers). Named after the project, like the Project itself.
// Returns null when there is nothing to configure.
export const projectConfigManifest = (state: WizardState): Manifest | null => {
  if (state.policies.length === 0) {
    return null;
  }
  return {
    apiVersion: 'kargo.akuity.io/v1alpha1',
    kind: 'ProjectConfig',
    metadata: { name: state.basics.name, namespace: state.basics.name },
    spec: { promotionPolicies: state.policies.map(promotionPolicySpec) }
  };
};

// All manifests the wizard will create, in creation order. Grows as more
// steps are implemented.
export const orderedManifests = (state: WizardState): Manifest[] => {
  if (!isValidProjectName(state.basics.name)) {
    return [];
  }
  const projectConfig = projectConfigManifest(state);
  return [
    projectManifest(state.basics),
    ...state.credentials
      .map((c) => credentialSecretManifest(c, state.basics.name))
      .filter((m): m is Manifest => m !== null),
    ...state.warehouses.map((w) => warehouseManifest(w, state.basics.name)),
    ...state.stages.map((s) => stageManifest(s, state.basics.name)),
    ...(projectConfig ? [projectConfig] : [])
  ];
};

export const resourceList = (state: WizardState): ResourceRef[] =>
  orderedManifests(state).map((m) => ({ kind: m.kind, name: m.metadata.name }));

export type CreationManifest = ResourceRef & { yaml: string };

// Per-resource manifests (with individual YAML bodies) in creation order —
// what the Step 6 creation engine applies one at a time.
export const creationManifests = (state: WizardState): CreationManifest[] =>
  orderedManifests(state).map((m) => ({
    kind: m.kind,
    name: m.metadata.name,
    yaml: yaml.stringify(m)
  }));

const placeholderComment = (lines: string[]) => lines.map((l) => `# ${l}`).join('\n') + '\n';

const toStringRecord = (value: unknown): Record<string, string> => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>).map(([k, v]) => [k, String(v)])
  );
};

// Inverse of projectManifest: maps an edited Project manifest back onto the
// basics slice. Throws with a user-facing message when the YAML is unusable.
export const basicsFromYaml = (text: string, prev: BasicsState): BasicsState => {
  const doc = yaml.parse(text);
  if (!doc || typeof doc !== 'object' || Array.isArray(doc)) {
    throw new Error('Expected a YAML mapping describing a Project');
  }
  if (doc.kind !== 'Project') {
    throw new Error('Expected kind: Project');
  }
  const metadata = (doc.metadata ?? {}) as Record<string, unknown>;
  const annotations = toStringRecord(metadata.annotations);
  return {
    ...prev,
    name: typeof metadata.name === 'string' ? metadata.name : '',
    description: annotations[annotationKeyDescription] ?? ''
  };
};

const credentialFromSecret = (doc: Record<string, unknown>): CredentialData => {
  const metadata = (doc.metadata ?? {}) as Record<string, unknown>;
  const labels = toStringRecord(metadata.labels);
  const annotations = toStringRecord(metadata.annotations);
  const data = toStringRecord(doc.stringData);

  const type = labels[labelKeyCredentialType] as CredentialType | undefined;
  if (!type || !credTypes.includes(type)) {
    throw new Error(
      `Secret ${metadata.name ?? '(unnamed)'} needs label ` +
        `${labelKeyCredentialType}: git | image | helm`
    );
  }

  const cred = initialCredential(type);
  cred.name = typeof metadata.name === 'string' ? metadata.name : '';
  cred.description = annotations[annotationKeyDescription] ?? '';

  cred.repoURL = data.repoURL ?? '';
  cred.repoURLIsRegex = data.repoURLIsRegex === 'true';
  if (
    data.githubAppClientID ||
    data.githubAppID ||
    data.githubAppInstallationID ||
    data.githubAppPrivateKey
  ) {
    cred.auth = 'github-app';
    cred.githubAppClientID = data.githubAppClientID ?? '';
    cred.githubAppID = data.githubAppID ?? '';
    cred.githubAppInstallationID = data.githubAppInstallationID ?? '';
    cred.githubAppPrivateKey = data.githubAppPrivateKey ?? '';
  } else if (data.sshPrivateKey) {
    cred.auth = 'ssh';
    cred.sshPrivateKey = data.sshPrivateKey;
  } else if (data.awsRegion || data.awsAccessKeyID || data.awsSecretAccessKey) {
    cred.auth = 'aws-ecr';
    cred.awsRegion = data.awsRegion ?? '';
    cred.awsAccessKeyID = data.awsAccessKeyID ?? '';
    cred.awsSecretAccessKey = data.awsSecretAccessKey ?? '';
  } else if (data.gcpServiceAccountKey) {
    cred.auth = 'gcp-ar';
    cred.gcpServiceAccountKey = tryDecodeBase64Utf8(data.gcpServiceAccountKey);
  } else {
    cred.auth = 'userpass';
    cred.username = data.username ?? '';
    cred.password = data.password ?? '';
  }
  return cred;
};

// Inverse of the credentials preview: maps edited Secret manifests (multi-doc)
// back onto the credentials slice. Ambient credentials have no manifest, so
// they carry over from the previous state untouched.
export const credentialsFromYaml = (text: string, prev: CredentialData[]): CredentialData[] => {
  const docs = yaml
    .parseAllDocuments(text)
    .map((d) => {
      if (d.errors.length > 0) {
        throw new Error(d.errors[0].message);
      }
      return d.toJS() as unknown;
    })
    .filter((d) => d !== null && d !== undefined);

  const parsed = docs.map((doc) => {
    if (!doc || typeof doc !== 'object' || Array.isArray(doc)) {
      throw new Error('Expected YAML mappings describing Secrets');
    }
    const record = doc as Record<string, unknown>;
    if (record.kind !== 'Secret') {
      throw new Error('Expected kind: Secret');
    }
    return credentialFromSecret(record);
  });

  return [...parsed, ...prev.filter((c) => c.auth === 'ambient')];
};

// Inverse of the warehouses preview: maps edited Warehouse manifests back onto
// the warehouses slice. Throws with a user-facing message on unusable input.
export const warehousesFromYaml = (text: string): WarehouseDraft[] => {
  const docs = yaml
    .parseAllDocuments(text)
    .map((d) => {
      if (d.errors.length > 0) {
        throw new Error(d.errors[0].message);
      }
      return d.toJS() as unknown;
    })
    .filter((d) => d !== null && d !== undefined);

  return docs.map((doc) => {
    if (!doc || typeof doc !== 'object' || Array.isArray(doc)) {
      throw new Error('Expected YAML mappings describing Warehouses');
    }
    const record = doc as Record<string, unknown>;
    if (record.kind !== 'Warehouse') {
      throw new Error('Expected kind: Warehouse');
    }
    const metadata = (record.metadata ?? {}) as Record<string, unknown>;
    const spec =
      record.spec && typeof record.spec === 'object'
        ? (record.spec as Record<string, unknown>)
        : { subscriptions: [] };
    return { name: typeof metadata.name === 'string' ? metadata.name : '', spec };
  });
};

// YAML shown in the live preview rail for a step.
export const yamlForStep = (state: WizardState, step: StepId): string => {
  switch (step) {
    case 'basics':
      if (!state.basics.name) {
        return placeholderComment([
          'Name your project to see its manifest.',
          '',
          'apiVersion: kargo.akuity.io/v1alpha1',
          'kind: Project'
        ]);
      }
      return yaml.stringify(projectManifest(state.basics));
    case 'credentials': {
      const manifests = state.credentials
        .map((c) => credentialSecretManifest(c, state.basics.name))
        .filter((m): m is Manifest => m !== null);
      const ambient = state.credentials.filter((c) => c.auth === 'ambient');
      if (manifests.length === 0 && ambient.length === 0) {
        return placeholderComment([
          'Nothing configured in this step yet.',
          'Credential Secrets you add will appear as YAML.'
        ]);
      }
      const docs = manifests.map((m) => yaml.stringify(m)).join('---\n');
      const ambientNote =
        ambient.length > 0
          ? placeholderComment(
              ambient.map((c) => `${c.name || '(unnamed)'}: ambient — no Secret is created`)
            )
          : '';
      return [docs, ambientNote].filter(Boolean).join(docs ? '---\n' : '');
    }
    case 'warehouses': {
      if (state.warehouses.length === 0) {
        return placeholderComment([
          'Nothing configured in this step yet.',
          'Warehouses you add will appear as YAML.'
        ]);
      }
      return state.warehouses
        .map((w) => yaml.stringify(warehouseManifest(w, state.basics.name)))
        .join('---\n');
    }
    case 'stages': {
      if (state.stages.length === 0) {
        return placeholderComment([
          'Nothing configured in this step yet.',
          'Stages you add will appear as YAML.'
        ]);
      }
      return state.stages
        .map((s) => yaml.stringify(stageManifest(s, state.basics.name)))
        .join('---\n');
    }
    case 'policies': {
      const projectConfig = projectConfigManifest(state);
      if (!projectConfig) {
        return placeholderComment([
          'Nothing configured in this step yet.',
          'Promotion policies you add will appear in a ProjectConfig here.'
        ]);
      }
      return yaml.stringify(projectConfig);
    }
    case 'review': {
      const manifests = orderedManifests(state);
      if (manifests.length === 0) {
        return placeholderComment(['Nothing to create yet.']);
      }
      return manifests.map((m) => yaml.stringify(m)).join('---\n');
    }
    default:
      return placeholderComment([
        'Nothing configured in this step yet.',
        'Resources you add here will appear as YAML.'
      ]);
  }
};
