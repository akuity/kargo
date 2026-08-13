import { StepId, WizardState, isValidProjectName } from './types';

export type StepMeta = {
  id: StepId;
  num: number;
  title: string;
  required: boolean;
  hint: string;
  intro: string;
  // Optional link to the docs page covering this step's concepts.
  docs?: string;
  // Step is locked (not clickable) until this state slice is non-empty.
  needs?: 'stages';
};

const DOCS_BASE = 'https://docs.kargo.io';

export const STEP_META: StepMeta[] = [
  {
    id: 'basics',
    num: 1,
    title: 'Project basics',
    required: true,
    hint: 'Name & namespace',
    intro:
      "A Kargo Project maps 1:1 to a Kubernetes Namespace. It's the boundary for all the resources you'll create in the next steps.",
    docs: `${DOCS_BASE}/user-guide/how-to-guides/working-with-projects`
  },
  {
    id: 'credentials',
    num: 2,
    title: 'Credentials',
    required: false,
    hint: 'Git / image / Helm',
    intro:
      'Store repository credentials as Secrets in the project namespace so warehouses and promotion steps can authenticate.',
    docs: `${DOCS_BASE}/operator-guide/security/managing-secrets`
  },
  {
    id: 'warehouses',
    num: 3,
    title: 'Warehouses',
    required: false,
    hint: 'Freight sources',
    intro:
      'Warehouses subscribe to container image, Git, and Helm chart repositories and assemble new artifacts into Freight.',
    docs: `${DOCS_BASE}/user-guide/how-to-guides/working-with-warehouses`
  },
  {
    id: 'stages',
    num: 4,
    title: 'Stages & pipeline',
    required: false,
    hint: 'Promotion graph',
    intro:
      'Stages define where Freight can be promoted, and the ordered promotion steps that run on each promotion.',
    docs: `${DOCS_BASE}/user-guide/how-to-guides/working-with-stages`
  },
  {
    id: 'policies',
    num: 5,
    title: 'Promotion policies',
    required: false,
    hint: 'Auto-promotion rules',
    intro: 'Enable automatic promotion of new Freight into selected stages.',
    docs: `${DOCS_BASE}/user-guide/how-to-guides/working-with-projects#promotion-policies`,
    needs: 'stages'
  },
  {
    id: 'review',
    num: 6,
    title: 'Review & create',
    required: true,
    hint: 'Apply manifests',
    intro: 'Review every resource the wizard will create, then apply them in order.'
  }
];

export const stepMetaById = (id: StepId): StepMeta =>
  STEP_META.find((s) => s.id === id) ?? STEP_META[0];

export const stepIndex = (id: StepId) => STEP_META.findIndex((s) => s.id === id);

export const isStepLocked = (meta: StepMeta, state: WizardState) =>
  !!meta.needs && state[meta.needs].length === 0;

// A step is "completed" when it holds data — derived from state, not from
// navigating past it. The terminal review step is never marked complete here.
export const isStepComplete = (id: StepId, state: WizardState): boolean => {
  switch (id) {
    case 'basics':
      return isValidProjectName(state.basics.name);
    case 'credentials':
      return state.credentials.length > 0;
    case 'warehouses':
      return state.warehouses.length > 0;
    case 'stages':
      return state.stages.length > 0;
    case 'policies':
      return state.policies.length > 0;
    default:
      return false;
  }
};
