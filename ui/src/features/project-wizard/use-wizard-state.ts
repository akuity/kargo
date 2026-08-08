import { useCallback, useMemo } from 'react';
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
  normalizeWarehouse
} from './types';

const draftKey = 'kargo.project-wizard.draft.v1';

type WizardDraft = {
  state: WizardState;
};

const isStepId = (id: string | null): id is StepId => !!id && STEP_META.some((s) => s.id === id);

export const useWizardState = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [draft, setDraft] = useLocalStorage<WizardDraft | undefined>(draftKey);

  // Merge the persisted draft over the initial shape so drafts written by an
  // older version of the wizard don't leave slices undefined.
  const state = useMemo<WizardState>(() => {
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
  }, [draft]);

  // Whether a persisted draft holds real data (used to offer resume vs. start
  // fresh on entry). Recomputes with the draft; callers snapshot it at mount.
  const hasSavedDraft = useMemo(() => {
    const d = draft?.state;
    return (
      !!d &&
      (!!d.basics?.name ||
        (d.credentials?.length ?? 0) > 0 ||
        (d.warehouses?.length ?? 0) > 0 ||
        (d.stages?.length ?? 0) > 0 ||
        (d.policies?.length ?? 0) > 0)
    );
  }, [draft]);

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
    (patch: Partial<WizardState>) => setDraft({ state: { ...state, ...patch } }),
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

  const reset = useCallback(() => setDraft(undefined), [setDraft]);

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
