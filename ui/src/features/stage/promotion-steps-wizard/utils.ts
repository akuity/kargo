import { RunnerWithConfiguration } from './types';

export const isRunnersEqual = (r1?: RunnerWithConfiguration, r2?: RunnerWithConfiguration) =>
  r1?.identifier === r2?.identifier && r1?.as === r2?.as;
