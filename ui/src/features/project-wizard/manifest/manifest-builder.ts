import yaml from 'yaml';
import { z } from 'zod';

import { CredentialTypeLabelKey } from '@ui/features/common/settings/secrets/types';
import { DESCRIPTION_ANNOTATION_KEY } from '@ui/features/common/utils';
import {
  Project,
  ProjectConfig,
  PromotionPolicy,
  Stage,
  V1ObjectMeta,
  V1Secret,
  Warehouse
} from '@ui/gen/api/v2/models';
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
  credentialSecretFields,
  credentialTypes,
  initialCredential,
  isValidProjectName,
  stringOrEmpty,
  stringRecordSchema,
  warehouseSpecSchema
} from '../types';

export type ResourceRef = {
  kind: string;
  name: string;
};

// The generated models describe these resources as read from the API, where
// every field is optional. The wizard writes them, so require what it always
// sets: kind and metadata.name key every resource through creation and retry.
type Manifest<T> = T & {
  apiVersion: string;
  kind: string;
  metadata: V1ObjectMeta & { name: string };
};

type AnyManifest = Manifest<Project | V1Secret | Warehouse | Stage | ProjectConfig>;

const base64Utf8 = (value: string) => btoa(String.fromCharCode(...new TextEncoder().encode(value)));

const tryDecodeBase64Utf8 = (value: string) => {
  try {
    return new TextDecoder().decode(Uint8Array.from(atob(value), (c) => c.charCodeAt(0)));
  } catch (_) {
    return value;
  }
};

// Placeholder shown in the YAML preview in place of real secret values. On
// round-trip (editing the credentials YAML) it means "keep the existing value".
export const MASKED_SECRET = '<hidden>';

const secretDataKeys = new Set<string>(credentialSecretFields);

// Masks secret values in a Secret manifest's stringData, for the YAML preview
// only — never applied on the creation path, so the API still receives the real
// secrets. Empty values stay empty, so it's clear when nothing is set.
const maskSecretManifest = (m: AnyManifest): AnyManifest => {
  if (!('stringData' in m) || !m.stringData) {
    return m;
  }
  const stringData = Object.fromEntries(
    Object.entries(m.stringData).map(([key, value]) =>
      secretDataKeys.has(key) && value ? [key, MASKED_SECRET] : [key, value]
    )
  );
  return { ...m, stringData };
};

export const projectManifest = (basics: BasicsState): Manifest<Project> => {
  const annotations: Record<string, string> = {};
  if (basics.description) {
    annotations[DESCRIPTION_ANNOTATION_KEY] = basics.description;
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
): Manifest<V1Secret> | null => {
  if (cred.auth === 'ambient') {
    return null;
  }
  return {
    apiVersion: 'v1',
    kind: 'Secret',
    metadata: {
      name: cred.name,
      namespace: project,
      labels: { [CredentialTypeLabelKey]: cred.type },
      ...(cred.description && {
        annotations: { [DESCRIPTION_ANNOTATION_KEY]: cred.description }
      })
    },
    stringData: credentialStringData(cred)
  };
};

// Mirrors warehouseManifestsGen in features/utils/manifest-generator. Clones
// the spec so cleanEmptyObjectValues (which mutates) never touches wizard state.
export const warehouseManifest = (draft: WarehouseDraft, project: string): Manifest<Warehouse> => ({
  apiVersion: 'kargo.akuity.io/v1alpha1',
  kind: 'Warehouse',
  metadata: { name: draft.name, namespace: project },
  spec: cleanEmptyObjectValues(structuredClone(draft.spec))
});

// Mirrors stageFormToYAML in features/stage/create-stage. Steps map from the
// reused PromotionStepsWizard's RunnerWithConfiguration to Stage promotion steps.
export const stageManifest = (draft: StageDraft, project: string): Manifest<Stage> => {
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

const promotionPolicySpec = (p: PolicyDraft): PromotionPolicy => {
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
export const projectConfigManifest = (state: WizardState): Manifest<ProjectConfig> | null => {
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
export const orderedManifests = (state: WizardState): AnyManifest[] => {
  if (!isValidProjectName(state.basics.name)) {
    return [];
  }
  const projectConfig = projectConfigManifest(state);
  return [
    projectManifest(state.basics),
    ...state.credentials
      .map((c) => credentialSecretManifest(c, state.basics.name))
      .filter((m): m is Manifest<V1Secret> => m !== null),
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

// --- Parsing hand-edited YAML back into wizard state ------------------------
// The input is whatever the user typed in the preview rail, so it is parsed,
// not cast: fields degrade via the lenient schemas shared with the persisted-
// draft parsers in ../types, and the rail shows a throw verbatim.

const metadataSchema = z
  .object({
    name: stringOrEmpty,
    labels: stringRecordSchema,
    annotations: stringRecordSchema
  })
  .catch({ name: '', labels: {}, annotations: {} });

// The schema-level error covers input that isn't a mapping at all -- an empty
// or comment-only document parses to null.
const manifestDocSchema = <K extends string>(kind: K) =>
  z.object(
    {
      kind: z.literal(kind, { error: `Expected kind: ${kind}` }),
      metadata: metadataSchema
    },
    { error: `Expected a YAML mapping describing a ${kind}` }
  );

const projectDocSchema = manifestDocSchema('Project');

const secretDocSchema = manifestDocSchema('Secret').extend({
  stringData: stringRecordSchema
});

// The spec's contents are validated by the RJSF form, which this path bypasses.
const warehouseDocSchema = manifestDocSchema('Warehouse').extend({
  spec: warehouseSpecSchema
});

type SecretDoc = z.infer<typeof secretDocSchema>;

// A ZodError's own `message` is a JSON dump of its issues. No field paths:
// every field above falls back, so only `kind` and the document itself can
// error, and both already name themselves.
const parseDoc = <S extends z.ZodType>(schema: S, doc: unknown): z.output<S> => {
  const result = schema.safeParse(doc);
  if (result.success) {
    return result.data;
  }
  throw new Error(result.error.issues.map((issue) => issue.message).join('; '));
};

// Surfaces YAML syntax errors and drops the empty documents a stray `---`
// leaves behind.
const parseDocs = <S extends z.ZodType>(schema: S, text: string): z.output<S>[] =>
  yaml
    .parseAllDocuments(text)
    .map((d) => {
      if (d.errors.length > 0) {
        throw new Error(d.errors[0].message);
      }
      return d.toJS() as unknown;
    })
    .filter((d) => d !== null && d !== undefined)
    .map((doc) => parseDoc(schema, doc));

// Inverse of projectManifest: maps an edited Project manifest back onto the
// basics slice. Throws with a user-facing message when the YAML is unusable.
export const basicsFromYaml = (text: string, prev: BasicsState): BasicsState => {
  const { metadata } = parseDoc(projectDocSchema, yaml.parse(text));
  return {
    ...prev,
    name: metadata.name,
    description: metadata.annotations[DESCRIPTION_ANNOTATION_KEY] ?? ''
  };
};

// Auth is inferred from which keys are present, not declared, so first match
// wins -- mirroring how pkg/credentials reads a Secret.
const credentialFromSecret = ({ metadata, stringData: data }: SecretDoc): CredentialData => {
  const type = metadata.labels[CredentialTypeLabelKey] as CredentialType | undefined;
  if (!type || !credentialTypes.includes(type)) {
    throw new Error(
      `Secret ${metadata.name || '(unnamed)'} needs label ` +
        `${CredentialTypeLabelKey}: git | image | helm`
    );
  }

  const cred = initialCredential(type);
  cred.name = metadata.name;
  cred.description = metadata.annotations[DESCRIPTION_ANNOTATION_KEY] ?? '';

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

// When the (masked) credentials preview is edited, secret fields still holding
// the MASKED_SECRET placeholder are restored from the matching previous
// credential (by name), so masking never overwrites a real secret. An unmatched
// masked field is left blank rather than persisting the placeholder.
const restoreMaskedSecrets = (cred: CredentialData, prev: CredentialData[]): CredentialData => {
  const previous = prev.find((c) => c.name === cred.name);
  const restored = { ...cred };
  for (const field of credentialSecretFields) {
    if (restored[field] === MASKED_SECRET) {
      restored[field] = previous?.[field] ?? '';
    }
  }
  return restored;
};

// Inverse of the credentials preview: maps edited Secret manifests (multi-doc)
// back onto the credentials slice. Ambient credentials have no manifest, so
// they carry over from the previous state untouched.
export const credentialsFromYaml = (text: string, prev: CredentialData[]): CredentialData[] => {
  const parsed = parseDocs(secretDocSchema, text).map(credentialFromSecret);

  return [
    ...parsed.map((c) => restoreMaskedSecrets(c, prev)),
    ...prev.filter((c) => c.auth === 'ambient')
  ];
};

// Inverse of the warehouses preview: maps edited Warehouse manifests back onto
// the warehouses slice. Throws with a user-facing message on unusable input.
export const warehousesFromYaml = (text: string): WarehouseDraft[] =>
  parseDocs(warehouseDocSchema, text).map(({ metadata, spec }) => ({
    name: metadata.name,
    spec
  }));

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
        .filter((m): m is Manifest<V1Secret> => m !== null);
      const ambient = state.credentials.filter((c) => c.auth === 'ambient');
      if (manifests.length === 0 && ambient.length === 0) {
        return placeholderComment([
          'Nothing configured in this step yet.',
          'Credential Secrets you add will appear as YAML.'
        ]);
      }
      const docs = manifests.map((m) => yaml.stringify(maskSecretManifest(m))).join('---\n');
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
      return manifests.map((m) => yaml.stringify(maskSecretManifest(m))).join('---\n');
    }
    default:
      return placeholderComment([
        'Nothing configured in this step yet.',
        'Resources you add here will appear as YAML.'
      ]);
  }
};
