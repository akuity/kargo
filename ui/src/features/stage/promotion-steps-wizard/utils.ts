import { RunnerWithConfiguration } from './types';

// Mirrors the maxLength on PromotionStep.description in the Kargo API.
export const STEP_DESCRIPTION_MAX_LENGTH = 256;

// The API rejects an empty description (minLength is 1), so a blank one must be
// left out of the manifest rather than sent as an empty string.
export const stepDescriptionForManifest = (description?: string): string | undefined =>
  description?.trim() || undefined;

export const isRunnersEqual = (r1?: RunnerWithConfiguration, r2?: RunnerWithConfiguration) =>
  r1?.identifier === r2?.identifier && r1?.as === r2?.as;

// The runner form is one RJSF instance reused as the user switches between
// steps. RJSF treats an undefined formData as uncontrolled and falls back to
// its internal state, which still holds the previously edited step's config.
// A step with no config yet must therefore start from an empty object, or the
// previous step's values leak into it as soon as it is opened or edited.
export const runnerFormData = (runner: RunnerWithConfiguration): Record<string, unknown> =>
  runner.state ?? {};
