import { describe, expect, test } from 'vitest';

import { AnalysisRunRequest } from '@ui/gen/api/v2/models';

import { aggregateTargets } from './aggregate-targets';

describe('aggregateTargets', () => {
  test('returns empty spec/status for an undefined request', () => {
    const [spec, status] = aggregateTargets(undefined);
    expect(spec.metrics).toEqual([]);
    expect(status.metricResults).toEqual([]);
  });

  test('copies overall phase, message and timestamps from the request status', () => {
    const request: AnalysisRunRequest = {
      spec: {
        stageName: 'stage',
        requestedFreight: [],
        freightCollection: {},
        targets: [],
        verificationTemplate: {}
      },
      status: {
        phase: 'Successful',
        message: 'all good',
        startedAt: '2026-01-01T00:00:00Z',
        completedAt: '2026-01-01T00:05:00Z',
        analysisRuns: []
      }
    };
    const [, status] = aggregateTargets(request);
    expect(status.phase).toBe('Successful');
    expect(status.message).toBe('all good');
    expect(status.startedAt).toBe('2026-01-01T00:00:00Z');
    expect(status.completedAt).toBe('2026-01-01T00:05:00Z');
  });

  test('does not prefix metric names for a single target', () => {
    const request: AnalysisRunRequest = {
      spec: {
        stageName: 'stage',
        requestedFreight: [],
        freightCollection: {},
        targets: [],
        verificationTemplate: {}
      },
      status: {
        analysisRuns: [
          {
            targetName: 'target-a',
            status: { metricResults: [{ name: 'latency' }] }
          }
        ]
      }
    };
    const [spec, status] = aggregateTargets(request);
    expect(spec.metrics).toEqual([{ name: 'latency' }]);
    expect(status.metricResults?.map((m) => m.name)).toEqual(['latency']);
  });

  test('prefixes metric names with the target name for multiple targets', () => {
    const request: AnalysisRunRequest = {
      spec: {
        stageName: 'stage',
        requestedFreight: [],
        freightCollection: {},
        targets: [],
        verificationTemplate: {}
      },
      status: {
        analysisRuns: [
          {
            targetName: 'target-a',
            status: { metricResults: [{ name: 'latency' }] }
          },
          {
            targetName: 'target-b',
            status: { metricResults: [{ name: 'latency' }, { name: 'errors' }] }
          }
        ]
      }
    };
    const [spec, status] = aggregateTargets(request);
    const names = ['target-a: latency', 'target-b: latency', 'target-b: errors'];
    expect(spec.metrics?.map((m) => m.name)).toEqual(names);
    expect(status.metricResults?.map((m) => m.name)).toEqual(names);
  });

  test('sums run summaries across targets', () => {
    const request: AnalysisRunRequest = {
      spec: {
        stageName: 'stage',
        requestedFreight: [],
        freightCollection: {},
        targets: [],
        verificationTemplate: {}
      },
      status: {
        analysisRuns: [
          {
            targetName: 'target-a',
            status: { runSummary: { count: 2, successful: 1, failed: 1 } }
          },
          {
            targetName: 'target-b',
            status: { runSummary: { count: 3, successful: 2, error: 1 } }
          }
        ]
      }
    };
    const [, status] = aggregateTargets(request);
    expect(status.runSummary).toEqual({
      count: 5,
      error: 1,
      failed: 1,
      inconclusive: 0,
      successful: 3
    });
  });
});
