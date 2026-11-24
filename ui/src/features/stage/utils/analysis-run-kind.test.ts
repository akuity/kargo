import { describe, expect, test } from 'vitest';

import { analysisRunKindRun, isAnalysisRun } from './analysis-run-kind';

describe('isAnalysisRun', () => {
  test('returns false for an EE extension run kind', () => {
    expect(isAnalysisRun('kargo.akuity.io/v1alpha1, Kind=AnalysisRunRequest')).toBe(false);
  });

  test('returns true for a Rollouts AnalysisRun kind', () => {
    expect(isAnalysisRun(analysisRunKindRun)).toBe(true);
  });

  test('returns true for a legacy (empty) kind', () => {
    expect(isAnalysisRun('')).toBe(true);
  });

  test('returns true when kind is undefined', () => {
    expect(isAnalysisRun(undefined)).toBe(true);
  });
});
