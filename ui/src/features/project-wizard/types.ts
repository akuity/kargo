import { z } from 'zod';

import type { RunnerWithConfiguration } from '@ui/features/stage/promotion-steps-wizard/types';
import type { FreightRequest } from '@ui/gen/api/v2/models';

export type StepId = 'basics' | 'credentials' | 'warehouses' | 'stages' | 'policies' | 'review';

// --- Lenient schemas for untrusted input ------------------------------------
// Two paths take input the wizard did not produce: hand-edited YAML in the
// preview rail (see manifest-builder) and drafts restored from localStorage
// (see the normalize* functions below). Both must degrade rather than reject --
// the rail commits while the user is still typing, and a draft may have been
// written by an older version of the wizard. These are the shared pieces.

// A string, or '' for anything that isn't one.
export const stringOrEmpty = z.string().catch('');

// Kubernetes label and annotation values are always strings, so scalars are
// coerced. Anything that isn't a mapping yields {}.
export const stringRecordSchema = z.record(z.string(), z.coerce.string()).catch({});

// A Warehouse spec is handed onward as written, so only its mapping-ness is
// enforced. Sequences and scalars fall back to an empty subscription list.
export const warehouseSpecSchema = z.record(z.string(), z.unknown()).catch({ subscriptions: [] });

// An array whose items are trusted as-is. Item types come from generated models
// (FreightRequest) or the promotion-steps registry (RunnerWithConfiguration);
// re-describing them here would duplicate the generator, so only array-ness is
// checked -- exactly what the hand-written checks did.
const looseArray = <T>() =>
  z
    .array(z.unknown())
    .catch([])
    .transform((items) => items as T[]);

export type BasicsState = {
  name: string;
  description: string;
};

// The wizard's credentials step is scoped to repository credentials only —
// git, image, and Helm chart repos, matching what Warehouses need to pull
// artifacts. Generic secrets (accessed via secret() in promotion steps) are
// out of scope for guided setup; they can be added later from project settings.
const credentialTypeEnum = z.enum(['git', 'image', 'helm']);

export type CredentialType = z.infer<typeof credentialTypeEnum>;

// The runtime counterpart of the union above, for membership checks.
export const credentialTypes = credentialTypeEnum.options;

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

// Classifies every CredentialData field: `true` marks secret material that must
// never be written to localStorage or shown in the YAML preview. Because this is
// checked against Record<keyof CredentialData, boolean>, adding a new field to
// CredentialData will NOT compile until it is classified here — so a new secret
// can never silently bypass stripping/masking. `as const` keeps the literal
// true/false values so the secret list can be derived from them.
const credentialFieldIsSecret = {
  name: false,
  description: false,
  type: false,
  repoURL: false,
  repoURLIsRegex: false,
  auth: false,
  username: false,
  password: true,
  sshPrivateKey: true,
  githubAppClientID: false,
  githubAppID: false,
  githubAppInstallationID: false,
  githubAppPrivateKey: true,
  awsRegion: false,
  awsAccessKeyID: false,
  awsSecretAccessKey: true,
  gcpServiceAccountKey: true
} as const satisfies Record<keyof CredentialData, boolean>;

// The secret field names (both CredentialData fields and, by the same names,
// Secret stringData keys), derived from the classification above.
export type CredentialSecretField = {
  [K in keyof typeof credentialFieldIsSecret]: (typeof credentialFieldIsSecret)[K] extends true
    ? K
    : never;
}[keyof typeof credentialFieldIsSecret];

export const credentialSecretFields = (
  Object.keys(credentialFieldIsSecret) as (keyof typeof credentialFieldIsSecret)[]
).filter((field): field is CredentialSecretField => credentialFieldIsSecret[field]);

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
// The schema is the single source of the four selector kinds: the type is
// derived from it, and normalizePolicy reuses it to validate persisted drafts.
const policySelectorEnum = z.enum(['exact', 'regex', 'glob', 'labels']);

export type PolicySelectorType = z.infer<typeof policySelectorEnum>;

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

// Coerces a persisted draft to the BasicsState shape, so every slice of a
// restored draft is checked rather than spread in unvalidated.
const basicsDraftSchema = z
  .object({ name: stringOrEmpty, description: stringOrEmpty })
  .catch({ name: '', description: '' });

export const normalizeBasics = (raw: unknown): BasicsState => basicsDraftSchema.parse(raw);

export const initialWizardState = (): WizardState => ({
  basics: initialBasicsState(),
  credentials: [],
  warehouses: [],
  stages: [],
  policies: []
});

// Returns a copy of the state with all credential secret material blanked, for
// safe persistence to localStorage. Secrets are kept in memory for the session
// only; a resumed draft's secret fields are empty and must be re-entered.
export const stripCredentialSecrets = (state: WizardState): WizardState => ({
  ...state,
  credentials: state.credentials.map((cred) => {
    const stripped = { ...cred };
    for (const field of credentialSecretFields) {
      stripped[field] = '';
    }
    return stripped;
  })
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
const warehouseDraftSchema = z
  .object({ name: stringOrEmpty, spec: warehouseSpecSchema })
  .catch({ name: '', spec: { subscriptions: [] } });

export const normalizeWarehouse = (raw: unknown): WarehouseDraft => warehouseDraftSchema.parse(raw);

// Coerces a persisted draft to the StageDraft shape (defensive, like the others).
const stageDraftSchema = z
  .object({
    name: stringOrEmpty,
    color: z.string().optional().catch(undefined),
    requestedFreight: looseArray<FreightRequest>(),
    steps: looseArray<RunnerWithConfiguration>()
  })
  .catch({ name: '', color: undefined, requestedFreight: [], steps: [] });

export const normalizeStage = (raw: unknown): StageDraft => stageDraftSchema.parse(raw);

// Coerces a persisted draft to the PolicyDraft shape. autoPromotionEnabled is
// coerced rather than type-checked, preserving the truthiness the hand-written
// version applied (`!!draft.autoPromotionEnabled`).
const policyDraftSchema = z
  .object({
    selectorType: policySelectorEnum.catch('exact'),
    value: stringOrEmpty,
    matchLabels: stringRecordSchema,
    autoPromotionEnabled: z.coerce.boolean().catch(false)
  })
  .catch({ selectorType: 'exact', value: '', matchLabels: {}, autoPromotionEnabled: false });

export const normalizePolicy = (raw: unknown): PolicyDraft => policyDraftSchema.parse(raw);

// RFC 1123 label: the Project name doubles as the Namespace name.
export const projectNameRegex = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;

export const isValidProjectName = (name: string) =>
  name.length > 0 && name.length <= 63 && projectNameRegex.test(name);
