import { RunnerWithConfiguration } from './types';

export const isRunnersEqual = (r1?: RunnerWithConfiguration, r2?: RunnerWithConfiguration) =>
  r1?.identifier === r2?.identifier && r1?.as === r2?.as;

// The runner form is one RJSF instance reused as the user switches between
// steps. RJSF treats an undefined formData as uncontrolled and falls back to
// its internal state, which still holds the previously edited step's config.
// A step with no config yet must therefore start from an empty object, or the
// previous step's values leak into it as soon as it is opened or edited.
export const runnerFormData = (runner: RunnerWithConfiguration): Record<string, unknown> =>
  runner.state ?? {};
