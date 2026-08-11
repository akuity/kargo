import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { useLocalStorage } from '@ui/utils/use-local-storage';

import { STEP_META, isStepLocked, stepIndex, stepMetaById } from './step-meta';
import {
  StepId,
  WizardState,
  initialWizardState,
  normalizeCredential,
  normalizePolicy,
  normalizeStage,
  normalizeWarehouse,
  stripCredentialSecrets
} from './types';

const draftKey = 'kargo.project-wizard.draft.v1';

type WizardDraft = {
  state: WizardState;
};

const isStepId = (id: string | null): id is StepId => !!id && STEP_META.some((s) => s.id === id);

// Merge a persisted draft over the initial shape so drafts written by an older
// version of the wizard don't leave slices undefined. Persisted drafts never
// contain secret material (see stripCredentialSecrets), so a resumed draft's
// credential secret fields are blank.
const hydrate = (draft: WizardDraft | undefined): WizardState => {
  const initial = initialWizardState();
  return {
    ...initial,
    ...draft?.state,
    basics: { ...initial.basics, ...draft?.state?.basics },
    credentials: (draft?.state?.credentials ?? []).map(normalizeCredential),
    warehouses: (draft?.state?.warehouses ?? []).map(normalizeWarehouse),
    stages: (draft?.state?.stages ?? []).map(normalizeStage),
    policies: (draft?.state?.policies ?? []).map(normalizePolicy)
  };
};

const draftHasRealData = (state: WizardState | undefined): boolean =>
  !!state &&
  (!!state.basics?.name ||
    (state.credentials?.length ?? 0) > 0 ||
    (state.warehouses?.length ?? 0) > 0 ||
    (state.stages?.length ?? 0) > 0 ||
    (state.policies?.length ?? 0) > 0);

export const useWizardState = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [draft, setDraft] = useLocalStorage<WizardDraft | undefined>(draftKey);

  // The working state lives in memory, hydrated once from the persisted draft.
  // Credential secrets are never persisted (patchState strips them before
  // writing), so they exist only here for the lifetime of the session.
  const [state, setState] = useState<WizardState>(() => hydrate(draft));

  // Whether a persisted draft holds real data (used to offer resume vs. start
  // fresh on entry). Recomputes with the draft; callers snapshot it at mount.
  const hasSavedDraft = useMemo(() => draftHasRealData(draft?.state), [draft]);

  const stepParam = searchParams.get('step');
  const current: StepId = isStepId(stepParam) ? stepParam : 'basics';

  const navigate = useCallback(
    (id: StepId) => setSearchParams(id === 'basics' ? {} : { step: id }),
    [setSearchParams]
  );

  // Sidebar jumps are gated on locked steps; Back / Continue are not — a locked
  // step still renders (with a locked body) when reached in sequence.
  const goTo = useCallback(
    (id: StepId) => {
      if (isStepLocked(stepMetaById(id), state)) {
        return;
      }
      navigate(id);
    },
    [navigate, state]
  );

  const patchState = useCallback(
    (patch: Partial<WizardState>) => {
      const next = { ...state, ...patch };
      setState(next);
      // Persist a copy with secrets removed — localStorage must never hold them.
      setDraft({ state: stripCredentialSecrets(next) });
    },
    [setDraft, state]
  );

  // Advance to the next step. Completion is derived from data, not tracked here.
  const advance = useCallback(() => {
    const next = STEP_META[Math.min(stepIndex(current) + 1, STEP_META.length - 1)];
    navigate(next.id);
  }, [current, navigate]);

  const back = useCallback(() => {
    const prev = STEP_META[Math.max(stepIndex(current) - 1, 0)];
    navigate(prev.id);
  }, [current, navigate]);

  const reset = useCallback(() => {
    setDraft(undefined);
    setState(initialWizardState());
  }, [setDraft]);

  return {
    state,
    hasSavedDraft,
    current,
    goTo,
    patchState,
    advance,
    back,
    reset
  };
};
