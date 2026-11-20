import { describe, expect, test } from 'vitest';

import { RolloutsAnalysisRun } from '@ui/gen/api/v2/models';

import { extractFilters, resolveSelection } from './extract-analysis-run';

const analysisRun = {
  spec: {
    metrics: [
      {
        name: 'test-suite',
        provider: {
          job: {
            spec: { template: { spec: { containers: [{ name: 'tests' }, { name: 'sidecar' }] } } }
          }
        }
      },
      {
        name: 'smoke',
        provider: { job: { spec: { template: { spec: { containers: [{ name: 'curl' }] } } } } }
      }
    ]
  }
} as RolloutsAnalysisRun;

describe('resolveSelection', () => {
  const items = extractFilters(analysisRun);

  test('defaults to the first metric and its first container', () => {
    expect(resolveSelection(items, {})).toEqual({
      selectedJob: 'test-suite',
      selectedContainer: 'tests'
    });
  });

  test('keeps a valid requested selection', () => {
    expect(
      resolveSelection(items, { selectedJob: 'test-suite', selectedContainer: 'sidecar' })
    ).toEqual({
      selectedJob: 'test-suite',
      selectedContainer: 'sidecar'
    });
  });

  test("falls back to the metric's first container when the container is unknown", () => {
    expect(resolveSelection(items, { selectedJob: 'smoke', selectedContainer: 'tests' })).toEqual({
      selectedJob: 'smoke',
      selectedContainer: 'curl'
    });
  });

  // A Full Screen link built before the AnalysisRun loaded carried
  // ?job=undefined&container=undefined, which must not win over the defaults.
  test('ignores the literal string "undefined" from URL params', () => {
    expect(
      resolveSelection(items, { selectedJob: 'undefined', selectedContainer: 'undefined' })
    ).toEqual({ selectedJob: 'test-suite', selectedContainer: 'tests' });
  });

  test('resolves once the AnalysisRun arrives after the initial render', () => {
    const requested = { selectedJob: undefined, selectedContainer: undefined };

    expect(resolveSelection(extractFilters(undefined), requested)).toEqual({
      selectedJob: undefined,
      selectedContainer: undefined
    });
    expect(resolveSelection(extractFilters(analysisRun), requested)).toEqual({
      selectedJob: 'test-suite',
      selectedContainer: 'tests'
    });
  });
});
