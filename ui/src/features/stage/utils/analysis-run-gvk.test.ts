import { describe, expect, test } from 'vitest';

import { analysisRunGVKRequest, analysisRunGVKRun, isAnalysisRunRequest } from './analysis-run-gvk';

describe('isAnalysisRunRequest', () => {
  test('returns true for an AnalysisRunRequest gvk', () => {
    expect(isAnalysisRunRequest(analysisRunGVKRequest)).toBe(true);
  });

  test('returns false for a Rollouts AnalysisRun gvk', () => {
    expect(isAnalysisRunRequest(analysisRunGVKRun)).toBe(false);
  });

  test('returns false for a legacy (empty) gvk', () => {
    expect(isAnalysisRunRequest('')).toBe(false);
  });

  test('returns false when gvk is undefined', () => {
    expect(isAnalysisRunRequest(undefined)).toBe(false);
  });
});
