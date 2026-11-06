import { describe, expect, test } from 'vitest';

import { RolloutsAnalysisRun, RolloutsMetricResult } from '@ui/gen/api/v2/models';

import {
  hasPendingJob,
  isJobPending,
  pendingJobRefetchIntervalMs,
  refetchWhileJobPending
} from './extract-analysis-run';

const jobMetric = (name: string) => ({
  name,
  provider: { job: { spec: { template: { spec: { containers: [{ name: 'main' }] } } } } }
});

const analysisRun = (
  phase: string | undefined,
  metricResults: RolloutsMetricResult[] = []
): RolloutsAnalysisRun => ({
  spec: {
    metrics: [
      jobMetric('job-a'),
      jobMetric('job-b'),
      { name: 'prometheus', provider: { prometheus: {} } }
    ]
  },
  status: { phase, metricResults }
});

const started = (name: string): RolloutsMetricResult => ({
  name,
  measurements: [{ metadata: { 'job-name': `${name}-1`, 'job-namespace': 'kargo-demo' } }]
});

describe('isJobPending', () => {
  test('no AnalysisRun', () => {
    expect(isJobPending(undefined, 'job-a')).toBe(false);
  });

  test('no metric name', () => {
    expect(isJobPending(analysisRun('Running'), undefined)).toBe(false);
  });

  test.each(['Successful', 'Failed', 'Error', 'Inconclusive'])(
    'run is %s and the Job never started',
    (phase) => {
      expect(isJobPending(analysisRun(phase), 'job-a')).toBe(false);
    }
  );

  test.each([undefined, '', 'Pending', 'Running'])(
    'run phase is %j and metric has no result',
    (phase) => {
      expect(isJobPending(analysisRun(phase), 'job-a')).toBe(true);
    }
  );

  test('metric result has no measurements', () => {
    expect(
      isJobPending(analysisRun('Running', [{ name: 'job-a', measurements: [] }]), 'job-a')
    ).toBe(true);
  });

  test('measurement has no Job name metadata', () => {
    expect(
      isJobPending(
        analysisRun('Running', [{ name: 'job-a', measurements: [{ metadata: {} }] }]),
        'job-a'
      )
    ).toBe(true);
  });

  test('measurement has no Job namespace metadata', () => {
    expect(
      isJobPending(
        analysisRun('Running', [
          { name: 'job-a', measurements: [{ metadata: { 'job-name': 'job-a-1' } }] }
        ]),
        'job-a'
      )
    ).toBe(true);
  });

  // The API server finds a Job's logs using the first measurement only.
  test('only a later measurement records the Job', () => {
    expect(
      isJobPending(
        analysisRun('Running', [
          {
            name: 'job-a',
            measurements: [
              { phase: 'Error' },
              { metadata: { 'job-name': 'job-a-1', 'job-namespace': 'kargo-demo' } }
            ]
          }
        ]),
        'job-a'
      )
    ).toBe(true);
  });

  test('Job for another metric has started', () => {
    expect(isJobPending(analysisRun('Running', [started('job-b')]), 'job-a')).toBe(true);
  });

  test('Job has started', () => {
    expect(isJobPending(analysisRun('Running', [started('job-a')]), 'job-a')).toBe(false);
  });
});

describe('hasPendingJob', () => {
  test('no AnalysisRun', () => {
    expect(hasPendingJob(undefined)).toBe(false);
  });

  test('run is complete', () => {
    expect(hasPendingJob(analysisRun('Successful'))).toBe(false);
  });

  test('Job for one metric has not started', () => {
    expect(hasPendingJob(analysisRun('Running', [started('job-a')]))).toBe(true);
  });

  test('Jobs for all metrics have started', () => {
    expect(hasPendingJob(analysisRun('Running', [started('job-a'), started('job-b')]))).toBe(false);
  });
});

describe('refetchWhileJobPending', () => {
  test('query failed', () => {
    expect(
      refetchWhileJobPending({ state: { status: 'error', data: { data: analysisRun('Running') } } })
    ).toBe(false);
  });

  test('Job has not started', () => {
    expect(
      refetchWhileJobPending({
        state: { status: 'success', data: { data: analysisRun('Running') } }
      })
    ).toBe(pendingJobRefetchIntervalMs);
  });

  test('all Jobs have started', () => {
    expect(
      refetchWhileJobPending({
        state: {
          status: 'success',
          data: { data: analysisRun('Running', [started('job-a'), started('job-b')]) }
        }
      })
    ).toBe(false);
  });
});
