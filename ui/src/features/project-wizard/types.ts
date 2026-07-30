import type { RunnerWithConfiguration } from '@ui/features/stage/promotion-steps-wizard/types';
import type { FreightRequest } from '@ui/gen/api/v2/models';

export type StepId = 'basics' | 'credentials' | 'warehouses' | 'stages' | 'policies' | 'review';

export type BasicsState = {
  name: string;
  description: string;
};

// The wizard's credentials step is scoped to repository credentials only —
// git, image, and Helm chart repos, matching what Warehouses need to pull
// artifacts. Generic secrets (accessed via secret() in promotion steps) are
// out of scope for guided setup; they can be added later from project settings.
export type CredentialType = 'git' | 'image' | 'helm';

// Auth methods map to the secret data keys understood by pkg/credentials/*.
// 'ambient' stores no Secret at all (IRSA / workload identity on the
// controller pod).
export type CredentialAuthMethod =
  'userpass' | 'ssh' | 'github-app' | 'aws-ecr' | 'gcp-ar' | 'ambient';

export type CredentialData = {
  name: string;
  description: string;
  type: CredentialType;
  repoURL: string;
  repoURLIsRegex: boolean;
  auth: CredentialAuthMethod;
  username: string;
  password: string;
  sshPrivateKey: string;
  githubAppClientID: string;
  githubAppID: string;
  githubAppInstallationID: string;
  githubAppPrivateKey: string;
  awsRegion: string;
  awsAccessKeyID: string;
  awsSecretAccessKey: string;
  gcpServiceAccountKey: string;
};

// Per docs/docs/50-user-guide/50-security/30-managing-secrets.md: gcp-ar also
// applies to OCI Helm chart repositories hosted in Google Artifact Registry.
export const credentialAuthOptions: Record<CredentialType, CredentialAuthMethod[]> = {
  git: ['userpass', 'ssh', 'github-app'],
  image: ['userpass', 'aws-ecr', 'gcp-ar', 'ambient'],
  helm: ['userpass', 'gcp-ar', 'ambient']
};

export const initialCredential = (type: CredentialType = 'git'): CredentialData => ({
  name: '',
  description: '',
  type,
  repoURL: '',
  repoURLIsRegex: false,
  auth: credentialAuthOptions[type][0] ?? 'userpass',
  username: '',
  password: '',
  sshPrivateKey: '',
  githubAppClientID: '',
  githubAppID: '',
  githubAppInstallationID: '',
  githubAppPrivateKey: '',
  awsRegion: '',
  awsAccessKeyID: '',
  awsSecretAccessKey: '',
  gcpServiceAccountKey: ''
});

// A warehouse draft is exactly the controlled form state of the app's
// CreateWarehouseWizard: a name plus the dynamic (RJSF) Warehouse spec.
export type WarehouseDraft = {
  name: string;
  spec: Record<string, unknown>;
};

export const initialWarehouse = (): WarehouseDraft => ({
  name: '',
  spec: { subscriptions: [] }
});

// A ready-to-use starter inspired by akuity's kargo-simple example
// (github.com/akuity/kargo-simple); it watches the public guestbook image
// akuity publishes for its Kargo examples.
export const exampleWarehouse = (): WarehouseDraft => ({
  name: 'guestbook',
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

// A stage draft holds the controlled state of the reused stage-creation
// building blocks: name + color, the FreightRequest[] from RequestedFreightEditor,
// and the RunnerWithConfiguration[] from PromotionStepsWizard.
export type StageDraft = {
  name: string;
  color?: string;
  requestedFreight: FreightRequest[];
  steps: RunnerWithConfiguration[];
};

export const initialStage = (): StageDraft => ({
  name: '',
  requestedFreight: [],
  steps: []
});

// A dev -> staging -> prod pipeline inspired by akuity's kargo-simple example
// (github.com/akuity/kargo-simple), wired to the guestbook Warehouse. Returned
// with no steps; StepStages fills promotion steps from the registry at load
// time (a stage with no steps would be a control-flow stage, not a normal one).
export const exampleStages = (warehouseName = 'guestbook'): StageDraft[] => [
  {
    name: 'dev',
    color: 'red',
    requestedFreight: [
      { origin: { kind: 'Warehouse', name: warehouseName }, sources: { direct: true, stages: [] } }
    ],
    steps: []
  },
  {
    name: 'staging',
    color: 'amber',
    requestedFreight: [
      {
        origin: { kind: 'Warehouse', name: warehouseName },
        sources: { direct: false, stages: ['dev'] }
      }
    ],
    steps: []
  },
  {
    name: 'prod',
    color: 'violet',
    requestedFreight: [
      {
        origin: { kind: 'Warehouse', name: warehouseName },
        sources: { direct: false, stages: ['staging'] }
      }
    ],
    steps: []
  }
];

// A promotion policy targets a Stage (by exact name, regex/glob pattern, or
// label selector) and toggles auto-promotion. It maps to a ProjectConfig
// spec.promotionPolicies entry using stageSelector (the modern field; the
// deprecated top-level `stage` field is intentionally not used).
export type PolicySelectorType = 'exact' | 'regex' | 'glob' | 'labels';

export type PolicyDraft = {
  selectorType: PolicySelectorType;
  // Stage name (exact) or pattern (regex/glob). Unused when selectorType is 'labels'.
  value: string;
  matchLabels: Record<string, string>;
  autoPromotionEnabled: boolean;
};

export const initialPolicy = (stageName = ''): PolicyDraft => ({
  selectorType: 'exact',
  value: stageName,
  matchLabels: {},
  autoPromotionEnabled: true
});

export type WizardState = {
  basics: BasicsState;
  credentials: CredentialData[];
  warehouses: WarehouseDraft[];
  stages: StageDraft[];
  policies: PolicyDraft[];
};

export const initialBasicsState = (): BasicsState => ({
  name: '',
  description: ''
});

export const initialWizardState = (): WizardState => ({
  basics: initialBasicsState(),
  credentials: [],
  warehouses: [],
  stages: [],
  policies: []
});

// Backfills fields added after a draft was persisted, and coerces credentials
// left over from when the wizard also handled generic secrets back to a repo
// credential so older drafts keep loading.
export const normalizeCredential = (raw: unknown): CredentialData => {
  const cred = { ...initialCredential(), ...(raw as Partial<CredentialData>) };
  if (!credentialAuthOptions[cred.type]) {
    cred.type = 'git';
  }
  if (!credentialAuthOptions[cred.type].includes(cred.auth)) {
    cred.auth = credentialAuthOptions[cred.type][0];
  }
  return cred;
};

// Coerces a persisted draft to the WarehouseDraft shape (older drafts from
// when this slice was an unimplemented placeholder may hold arbitrary data).
export const normalizeWarehouse = (raw: unknown): WarehouseDraft => {
  const draft = (raw ?? {}) as Partial<WarehouseDraft>;
  return {
    name: typeof draft.name === 'string' ? draft.name : '',
    spec: draft.spec && typeof draft.spec === 'object' ? draft.spec : { subscriptions: [] }
  };
};

// Coerces a persisted draft to the StageDraft shape (defensive, like the others).
export const normalizeStage = (raw: unknown): StageDraft => {
  const draft = (raw ?? {}) as Partial<StageDraft>;
  return {
    name: typeof draft.name === 'string' ? draft.name : '',
    color: typeof draft.color === 'string' ? draft.color : undefined,
    requestedFreight: Array.isArray(draft.requestedFreight) ? draft.requestedFreight : [],
    steps: Array.isArray(draft.steps) ? draft.steps : []
  };
};

// Coerces a persisted draft to the PolicyDraft shape.
export const normalizePolicy = (raw: unknown): PolicyDraft => {
  const draft = (raw ?? {}) as Partial<PolicyDraft>;
  const selectorType: PolicySelectorType =
    draft.selectorType === 'regex' ||
    draft.selectorType === 'glob' ||
    draft.selectorType === 'labels'
      ? draft.selectorType
      : 'exact';
  return {
    selectorType,
    value: typeof draft.value === 'string' ? draft.value : '',
    matchLabels:
      draft.matchLabels && typeof draft.matchLabels === 'object' ? draft.matchLabels : {},
    autoPromotionEnabled: !!draft.autoPromotionEnabled
  };
};

// RFC 1123 label: the Project name doubles as the Namespace name.
export const projectNameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

export const isValidProjectName = (name: string) =>
  name.length > 0 && name.length <= 63 && projectNameRegex.test(name);
