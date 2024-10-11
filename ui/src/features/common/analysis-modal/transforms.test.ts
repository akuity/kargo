// eslint-disable-file @typescript-eslint/ban-ts-comment
import { describe, test, beforeAll, afterAll, expect } from 'vitest';

import {
  GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1Argument,
  GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1CloudWatchMetric,
  GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricProvider,
  GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricResult
} from '../../../models/rollout/generated';

import {
  analysisEndTime,
  analysisStartTime,
  argValue,
  chartMax,
  conditionDetails,
  formatKeyValueMeasurement,
  formatMultiItemArrayMeasurement,
  formatSingleItemArrayMeasurement,
  formatThresholdsForChart,
  formattedValue,
  interpolateQuery,
  isChartable,
  isValidDate,
  metricProvider,
  metricStatusLabel,
  metricSubstatus,
  printableCloudWatchQuery,
  printableDatadogQuery
} from './transforms';
import { AnalysisStatus, FunctionalStatus } from './types';

const MOCK_METRICS_WITHOUT_END_TIMES: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricResult[] =
  [
    {
      measurements: []
    },
    {
      measurements: [{}, {}]
    }
  ];

const MOCK_METRICS_WITH_END_TIMES: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricResult[] =
  [
    {
      measurements: [
        {
          // @ts-ignore
          finishedAt: '2023-11-16T00:25:23Z'
        },
        {
          // @ts-ignore
          finishedAt: '2023-11-16T00:26:23Z'
        },
        {
          // @ts-ignore
          finishedAt: '2023-11-16T00:27:23Z'
        },
        {
          // @ts-ignore
          finishedAt: '2023-11-16T00:28:23Z'
        }
      ]
    }
  ];

const MOCK_ARGS: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1Argument[] = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  },
  {
    name: 'application-name',
    value: 'istio-host-split-canary'
  },
  {
    name: 'cpu-usage-threshold'
  },
  {
    name: 'success-rate-threshold',
    value: '0.95'
  },
  {
    name: 'latency-threshold',
    value: '500'
  }
];

const MOCK_PROVIDER_PROMETHEUS: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricProvider =
  {
    prometheus: {
      address: 'https://prometheus-k8s.monitoring:9090',
      query:
        'sum(irate(istio_requests_total{destination_service_name=~"{{args.service-name}}",response_code!~"5.*"}[1m])) \n/\nsum(irate(istio_requests_total{destination_service_name=~"{{args.service-name}}"}[1m]))'
    }
  };
const MOCK_PROVIDER_NEWRELIC: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricProvider = {
  newRelic: {
    query:
      "FROM Transaction SELECT percentage(count(*), WHERE httpResponseCode != 500) as successRate where appName = '{{ args.application-name }}'"
  }
};
const MOCK_PROVIDER_DATADOG_V2_1: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricProvider =
  {
    datadog: {
      apiVersion: 'v2',
      query: 'sum:requests.errors{service:{{args.service-name}}}.as_count()',
      formula: "moving_rollup(a, 60, 'sum') / b"
    }
  };
const MOCK_PROVIDER_DATADOG_V2_2: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1MetricProvider =
  {
    datadog: {
      apiVersion: 'v2',
      queries: {
        a: 'sum:requests.errors{service:{{args.service-name}}}.as_count()',
        b: 'sum:requests{service:{{args.service-name}}}.as_count()'
      },
      formula: "moving_rollup(a, 60, 'sum') / b"
    }
  };

const MOCK_PROVIDER_CLOUDWATCH: GithubComArgoprojArgoRolloutsPkgApisRolloutsV1alpha1CloudWatchMetric =
  {
    metricDataQueries: [
      {
        id: 'rate',
        expression: 'errors / requests'
      },
      {
        id: 'errors',
        metricStat: {
          metric: {
            namespace: 'app',
            metricName: 'errors'
          },
          stat: 'Sum',
          unit: 'Count'
        },
        returnData: false
      },
      {
        id: 'requests',
        metricStat: {
          metric: {
            namespace: 'app',
            metricName: 'requests'
          },
          stat: 'Sum',
          unit: 'Count'
        },
        returnData: false
      }
    ]
  };

const MOCK_ARGS_PROMETHEUS = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  }
];
const MOCK_QUERY_PROMETHEUS =
  'sum(irate(istio_requests_total{reporter="source",destination_service=~"{{args.service-name}}",response_code!~"5.*"}[5m])) / sum(irate(istio_requests_total{reporter="source",destination_service=~"{{args.service-name}}"}[5m]))';

const MOCK_ARGS_NEWRELIC = [{ name: 'application-name', value: 'myApp' }];
const MOCK_QUERY_NEWRELIC =
  "FROM Transaction SELECT percentage(count(*), WHERE httpResponseCode != 500) as successRate where appName = '{{ args.application-name }}'";

const MOCK_ARGS_DATADOG = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  }
];
const MOCK_QUERY_DATADOG = 'sum:requests.error.rate{service:{{args.service-name}}}';

const MOCK_ARGS_WAVEFRONT = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  }
];
const MOCK_QUERY_WAVEFRONT =
  'sum(rate(5m, ts("istio.requestcount.count", response_code!=500 and destination_service="{{args.service-name}}"))) / sum(rate(5m, ts("istio.requestcount.count", reporter=client and destination_service="{{args.service-name}}")))';

const MOCK_QUERY_GRAPHITE =
  "target=summarize(asPercent(sumSeries(stats.timers.httpServerRequests.app.{{args.service-name}}.exception.*.method.*.outcome.{CLIENT_ERROR,INFORMATIONAL,REDIRECTION,SUCCESS}.status.*.uri.*.count), sumSeries(stats.timers.httpServerRequests.app.{{args.service-name}}.exception.*.method.*.outcome.*.status.*.uri.*.count)),'5min','avg')";
const MOCK_ARGS_GRAPHITE = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  }
];

const MOCK_QUERY_INFLUXDB =
  'from(bucket: "app_istio") range(start: -15m) filter(fn: (r) => r["destination_workload"] == "{{ args.application-name }}")|> filter(fn: (r) => r["_measurement"] == "istio:istio_requests_errors_percentage:rate1m:5xx")';
const MOCK_ARGS_INFLUXDB = [{ name: 'application-name', value: 'myApp' }];

const MOCK_QUERY_SKYWALKING =
  'query queryData($duration: Duration!) { service_apdex: readMetricsValues(condition: { name: "service_apdex", entity: { scope: Service, serviceName: "{{ args.service-name }}", normal: true } }, duration: $duration) { label values { values { value } } } }';
const MOCK_ARGS_SKYWALKING = [
  {
    name: 'service-name',
    value: 'istio-host-split-canary'
  }
];

const MOCK_CONDITION_1 = 'result[0] < .95';
const MOCK_CONDITION_2 = 'result[0] > .5 && result[0] < .95';
const MOCK_CONDITION_3 = 'result.successRate >= 0.95';
const MOCK_CONDITION_4 = 'result.successRate >= {{ args.success-rate-threshold }}';
const MOCK_CONDITION_5 =
  'result.successRate >= {{ args.success-rate-threshold }} && result.errorRate <= 0.1';

describe('analysis modal transforms', () => {
  beforeAll(() => {});
  afterAll(() => {});

  test('isValidDate() for undefined', () => {
    expect(isValidDate()).toBe(false);
  });
  test('isValidDate() for a non-date recognized string', () => {
    expect(isValidDate('abcd')).toBe(false);
  });
  test('isValidDate() for a date recognized string', () => {
    expect(isValidDate('2023-11-16T00:25:23Z')).toBe(true);
  });

  test('analysisStartTime() for undefined', () => {
    expect(analysisStartTime()).toBeNull();
  });
  test('analysisStartTime() for a non-date recognized string', () => {
    expect(analysisStartTime('abcd')).toBeNull();
  });
  test('analysisStartTime() for a date recognized string', () => {
    expect(analysisStartTime('2023-11-16T00:25:23Z')).toBe(1700094323000);
  });

  test('analysisEndTime() for no metric results', () => {
    expect(analysisEndTime([])).toBe(null);
  });
  test('analysisEndTime() for analysis with metrics but no measurements', () => {
    expect(analysisEndTime(MOCK_METRICS_WITHOUT_END_TIMES)).toBe(null);
  });
  test('analysisEndTime() for measurements with finishedAt times', () => {
    expect(analysisEndTime(MOCK_METRICS_WITH_END_TIMES)).toBe(1700094503000);
  });

  test('argValue() for empty args', () => {
    expect(argValue([], 'cpu-threhold')).toBeNull();
  });
  test('argValue() for missing arg name / value', () => {
    expect(argValue(MOCK_ARGS, 'memory-threshold')).toBeNull();
  });
  test('argValue() for missing arg value', () => {
    expect(argValue(MOCK_ARGS, 'cpu-usage-treshold')).toBeNull();
  });
  test('argValue() for present arg name / value', () => {
    expect(argValue(MOCK_ARGS, 'latency-threshold')).toBe('500');
  });

  test('metricProvider() for known provider', () => {
    expect(metricProvider(MOCK_PROVIDER_PROMETHEUS)).toBe('prometheus');
  });

  test('conditionDetails with missing condition', () => {
    expect(conditionDetails(undefined, MOCK_ARGS, MOCK_PROVIDER_PROMETHEUS)).toEqual({
      label: null,
      thresholds: [],
      conditionKeys: []
    });
  });
  test('conditionDetails() with missing provider', () => {
    expect(conditionDetails(MOCK_CONDITION_1, MOCK_ARGS)).toEqual({
      label: null,
      thresholds: [],
      conditionKeys: []
    });
  });
  test('conditionDetails() for unsupported format', () => {
    expect(conditionDetails('result in resultsArray', MOCK_ARGS, MOCK_PROVIDER_PROMETHEUS)).toEqual(
      {
        label: 'result in resultsArray',
        thresholds: [],
        conditionKeys: []
      }
    );
  });
  test('conditionDetails() with missing args', () => {
    expect(conditionDetails(MOCK_CONDITION_1, undefined, MOCK_PROVIDER_PROMETHEUS)).toEqual({
      conditionKeys: ['0'],
      label: 'result[0] < .95',
      thresholds: [0.95]
    });
  });
  test('conditionDetails() for condition like result[0] [>, <] [number]', () => {
    expect(conditionDetails(MOCK_CONDITION_1, MOCK_ARGS, MOCK_PROVIDER_PROMETHEUS)).toEqual({
      conditionKeys: ['0'],
      label: 'result[0] < .95',
      thresholds: [0.95]
    });
  });
  test('conditionDetails() for multiple conditions like result[0] [>, <] [number] && result[0] [>, <] [number]', () => {
    expect(conditionDetails(MOCK_CONDITION_2, MOCK_ARGS, MOCK_PROVIDER_PROMETHEUS)).toEqual({
      conditionKeys: ['0'],
      label: 'result[0] > .5 && result[0] < .95',
      thresholds: [0.5, 0.95]
    });
  });
  test('conditionDetails() for condition like result.[key] [>, <] [number]', () => {
    expect(conditionDetails(MOCK_CONDITION_3, MOCK_ARGS, MOCK_PROVIDER_NEWRELIC)).toEqual({
      conditionKeys: ['successRate'],
      label: 'result.successRate >= 0.95',
      thresholds: [0.95]
    });
  });
  test('conditionDetails() for condition like result.[key] [>, <] [arg value]', () => {
    expect(conditionDetails(MOCK_CONDITION_4, MOCK_ARGS, MOCK_PROVIDER_NEWRELIC)).toEqual({
      conditionKeys: ['successRate'],
      label: 'result.successRate >= 0.95',
      thresholds: [0.95]
    });
  });
  test('conditionDetails() for multiple condition like result.[key1] [>, <] [arg value] && result.[key2] [>, <] [number]', () => {
    expect(conditionDetails(MOCK_CONDITION_5, MOCK_ARGS, MOCK_PROVIDER_NEWRELIC)).toEqual({
      conditionKeys: ['successRate', 'errorRate'],
      label: 'result.successRate >= 0.95 && result.errorRate <= 0.1',
      thresholds: [0.95, 0.1]
    });
  });

  test('formatThresholdsForChart() with number values', () => {
    expect(formatThresholdsForChart([0, 1.1, 2.22, 3.333, 4.4444, 5.55555])).toEqual([
      0, 1.1, 2.22, 3.33, 4.44, 5.56
    ]);
  });

  test('chartMax() for 0 max value and null thresholds', () => {
    expect(chartMax(0, null, null)).toBe(1);
  });
  test('chartMax() for 1 max value and null thresholds', () => {
    expect(chartMax(1, null, null)).toBe(1.2);
  });
  test('chartMax() for max value and thresholds that are the same', () => {
    expect(chartMax(2, [2], [2])).toBe(2.4);
  });
  test('chartMax() for max value that is above thresholds', () => {
    expect(chartMax(4, [2, 3], [1, 2])).toBe(4.8);
  });
  test('chartMax() for fail threshold that is above value and success threshold', () => {
    expect(chartMax(2, [2, 3, 4], [1, 2])).toBe(4.8);
  });
  test('chartMax() for success threshold that is above value and fail threshold', () => {
    expect(chartMax(2, [2, 3, 4], [1, 2, 6])).toBe(7.2);
  });

  test('metricSubstatus() for metric with pending status', () => {
    expect(metricSubstatus(AnalysisStatus.Pending, 0, 0, 0)).toBe(undefined);
  });
  test('metricSubstatus() for successful metric with no issues', () => {
    expect(metricSubstatus(AnalysisStatus.Successful, 0, 0, 0)).toBe(undefined);
  });
  test('metricSubstatus() for successful metric with failures', () => {
    expect(metricSubstatus(AnalysisStatus.Successful, 2, 0, 0)).toBe(FunctionalStatus.ERROR);
  });
  test('metricSubstatus() for successful metric with errors', () => {
    expect(metricSubstatus(AnalysisStatus.Successful, 0, 2, 0)).toBe(FunctionalStatus.WARNING);
  });

  test('metricStatusLabel() for metric with unknown status', () => {
    expect(metricStatusLabel(AnalysisStatus.Unknown, 0, 0, 0)).toBe('Analysis status unknown');
  });
  test('metricStatusLabel() for metric with successful status with failures', () => {
    expect(metricStatusLabel(AnalysisStatus.Successful, 1, 0, 0)).toBe(
      'Analysis passed with measurement failures'
    );
  });
  test('metricStatusLabel() for metric with successful status with errors', () => {
    expect(metricStatusLabel(AnalysisStatus.Successful, 0, 1, 0)).toBe(
      'Analysis passed with measurement errors'
    );
  });
  test('metricStatusLabel() for metric with successful status with inconclusives', () => {
    expect(metricStatusLabel(AnalysisStatus.Successful, 0, 0, 1)).toBe(
      'Analysis passed with inconclusive measurements'
    );
  });
  test('metricStatusLabel() for metric with successful status with multiple issues', () => {
    expect(metricStatusLabel(AnalysisStatus.Successful, 1, 2, 3)).toBe(
      'Analysis passed with multiple issues'
    );
  });

  test('interpolateQuery() for no query', () => {
    expect(interpolateQuery(undefined, MOCK_ARGS)).toBe(undefined);
  });
  test('interpolateQuery() for prometheus query with no args', () => {
    expect(interpolateQuery(MOCK_QUERY_PROMETHEUS, [])).toBe(MOCK_QUERY_PROMETHEUS);
  });
  test('interpolateQuery() for prometheus query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_PROMETHEUS, MOCK_ARGS_PROMETHEUS)).toBe(
      'sum(irate(istio_requests_total{reporter="source",destination_service=~"istio-host-split-canary",response_code!~"5.*"}[5m])) / sum(irate(istio_requests_total{reporter="source",destination_service=~"istio-host-split-canary"}[5m]))'
    );
  });
  test('interpolateQuery() for newrelic query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_NEWRELIC, MOCK_ARGS_NEWRELIC)).toBe(
      "FROM Transaction SELECT percentage(count(*), WHERE httpResponseCode != 500) as successRate where appName = 'myApp'"
    );
  });
  test('interpolateQuery() for simple datadog query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_DATADOG, MOCK_ARGS_DATADOG)).toBe(
      'sum:requests.error.rate{service:istio-host-split-canary}'
    );
  });
  test('interpolateQuery() for wavefront query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_WAVEFRONT, MOCK_ARGS_WAVEFRONT)).toBe(
      'sum(rate(5m, ts("istio.requestcount.count", response_code!=500 and destination_service="istio-host-split-canary"))) / sum(rate(5m, ts("istio.requestcount.count", reporter=client and destination_service="istio-host-split-canary")))'
    );
  });
  test('interpolateQuery() for graphite query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_GRAPHITE, MOCK_ARGS_GRAPHITE)).toBe(
      "target=summarize(asPercent(sumSeries(stats.timers.httpServerRequests.app.istio-host-split-canary.exception.*.method.*.outcome.{CLIENT_ERROR,INFORMATIONAL,REDIRECTION,SUCCESS}.status.*.uri.*.count), sumSeries(stats.timers.httpServerRequests.app.istio-host-split-canary.exception.*.method.*.outcome.*.status.*.uri.*.count)),'5min','avg')"
    );
  });
  test('interpolateQuery() for influxdb query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_INFLUXDB, MOCK_ARGS_INFLUXDB)).toBe(
      'from(bucket: "app_istio") range(start: -15m) filter(fn: (r) => r["destination_workload"] == "myApp")|> filter(fn: (r) => r["_measurement"] == "istio:istio_requests_errors_percentage:rate1m:5xx")'
    );
  });
  test('interpolateQuery() for skywalking query and args', () => {
    expect(interpolateQuery(MOCK_QUERY_SKYWALKING, MOCK_ARGS_SKYWALKING)).toBe(
      'query queryData($duration: Duration!) { service_apdex: readMetricsValues(condition: { name: "service_apdex", entity: { scope: Service, serviceName: "istio-host-split-canary", normal: true } }, duration: $duration) { label values { values { value } } } }'
    );
  });

  test('printableDataDogQuery() with v2 query and formula', () => {
    expect(
      printableDatadogQuery(MOCK_PROVIDER_DATADOG_V2_1.datadog, MOCK_ARGS_DATADOG)
    ).toStrictEqual([
      `query: sum:requests.errors{service:istio-host-split-canary}.as_count(), formula: moving_rollup(a, 60, 'sum') / b`
    ]);
  });
  test('printableDataDogQuery() with v2 queries and formula', () => {
    expect(
      printableDatadogQuery(MOCK_PROVIDER_DATADOG_V2_2.datadog, MOCK_ARGS_DATADOG)
    ).toStrictEqual([
      `queries: {"a":"sum:requests.errors{service:istio-host-split-canary}.as_count()","b":"sum:requests{service:istio-host-split-canary}.as_count()"}, formula: moving_rollup(a, 60, 'sum') / b`
    ]);
  });

  test('printableCloudWatchQuery() with metricDataQueries', () => {
    expect(printableCloudWatchQuery(MOCK_PROVIDER_CLOUDWATCH)).toStrictEqual([
      '{"id":"rate","expression":"errors / requests"}',
      '{"id":"errors","metricStat":{"metric":{"namespace":"app","metricName":"errors"},"stat":"Sum","unit":"Count"},"returnData":false}',
      '{"id":"requests","metricStat":{"metric":{"namespace":"app","metricName":"requests"},"stat":"Sum","unit":"Count"},"returnData":false}'
    ]);
  });

  test('isChartable() for undefined', () => {
    expect(isChartable(undefined)).toBe(false);
  });
  test('isChartable() for null', () => {
    expect(isChartable(null)).toBe(true);
  });
  test('isChartable() for a string', () => {
    expect(isChartable('abc')).toBe(false);
  });
  test('isChartable() for an array', () => {
    expect(isChartable([1, 2, 5, 3])).toBe(false);
  });
  test('isChartable() for a positive number', () => {
    expect(isChartable(5)).toBe(true);
  });
  test('isChartable() for a negative number', () => {
    expect(isChartable(-5)).toBe(true);
  });

  test('formattedValue() for null', () => {
    expect(formattedValue(null)).toBe(null);
  });
  test('formattedValue() for an int', () => {
    expect(formattedValue(1)).toBe(1);
  });
  test('formattedValue() for a float', () => {
    expect(formattedValue(1.2653)).toBe(1.27);
  });
  test('formattedValue() for a string', () => {
    expect(formattedValue('abc')).toBe('abc');
  });
  test('formattedValue() for an array of numbers', () => {
    expect(formattedValue([1, 4, 3, 7])).toBe('1,4,3,7');
  });

  test('formatSingleItemArrayMeasurement() with out of bounds accessor', () => {
    expect(formatSingleItemArrayMeasurement([4], 1)).toEqual({
      canChart: true,
      chartValue: { 1: null },
      tableValue: { 1: null }
    });
  });
  test('formatSingleItemArrayMeasurement() for a value like [`abc`] with accessor 0', () => {
    expect(formatSingleItemArrayMeasurement(['abc'], 0)).toEqual({
      canChart: false,
      tableValue: { 0: 'abc' }
    });
  });
  test('formatSingleItemArrayMeasurement() for a value like [4] with accessor 0', () => {
    expect(formatSingleItemArrayMeasurement([4], 0)).toEqual({
      canChart: true,
      chartValue: { 0: 4 },
      tableValue: { 0: 4 }
    });
  });
  test('formatSingleItemArrayMeasurement() for a value like [null] with accessor 0', () => {
    expect(formatSingleItemArrayMeasurement([null], 0)).toEqual({
      canChart: true,
      chartValue: { 0: null },
      tableValue: { 0: null }
    });
  });

  test('formatMultiItemArrayMeasurement() with an empty array', () => {
    expect(formatMultiItemArrayMeasurement([])).toEqual({
      canChart: false,
      tableValue: ''
    });
  });
  test('formatMultiItemArrayMeasurement() with all numbers', () => {
    expect(formatMultiItemArrayMeasurement([4, 6, 3, 5])).toEqual({
      canChart: true,
      chartValue: 4,
      tableValue: '4,6,3,5'
    });
  });
  test('formatMultiItemArrayMeasurement() with null as the first item', () => {
    expect(formatMultiItemArrayMeasurement([null, 6, 3, 5])).toEqual({
      canChart: true,
      chartValue: null,
      tableValue: 'null,6,3,5'
    });
  });
  test('formatMultiItemArrayMeasurement() with a string as the first item', () => {
    expect(formatMultiItemArrayMeasurement(['abc', 6, 3, 5])).toEqual({
      canChart: false,
      tableValue: 'abc,6,3,5'
    });
  });

  test('formatKeyValueMeasurement() with key value pairs and no matching accessors', () => {
    expect(formatKeyValueMeasurement({ cpuUsage: 50, latency: 500 }, ['errorRate'])).toEqual({
      canChart: false,
      chartValue: { errorRate: null },
      tableValue: { errorRate: null }
    });
  });
  test('formatKeyValueMeasurement() with key value pairs and a single matching accessor', () => {
    expect(formatKeyValueMeasurement({ cpuUsage: 50, latency: 500 }, ['latency'])).toEqual({
      canChart: true,
      chartValue: { latency: 500 },
      tableValue: { latency: 500 }
    });
  });
  test('formatKeyValueMeasurement() with key value pairs and multiple matching accessors', () => {
    expect(
      formatKeyValueMeasurement({ cpuUsage: 50, latency: 500 }, ['latency', 'cpuUsage'])
    ).toEqual({
      canChart: true,
      chartValue: { latency: 500, cpuUsage: 50 },
      tableValue: { latency: 500, cpuUsage: 50 }
    });
  });
  test('formatKeyValueMeasurement() with key value pairs all null and matching accessors', () => {
    expect(
      formatKeyValueMeasurement({ cpuUsage: null, latency: null }, ['latency', 'cpuUsage'])
    ).toEqual({
      canChart: false,
      chartValue: { latency: null, cpuUsage: null },
      tableValue: { latency: null, cpuUsage: null }
    });
  });
});
