import { RunnerWithConfiguration } from './types';

// Mirrors the maxLength on PromotionStep.description in the Kargo API.
export const STEP_DESCRIPTION_MAX_LENGTH = 256;

// The API rejects an empty description (minLength is 1), so a blank one must be
// left out of the manifest rather than sent as an empty string.
export const stepDescriptionForManifest = (description?: string): string | undefined =>
  description?.trim() || undefined;

export const isRunnersEqual = (r1?: RunnerWithConfiguration, r2?: RunnerWithConfiguration) =>
  r1?.identifier === r2?.identifier && r1?.as === r2?.as;
