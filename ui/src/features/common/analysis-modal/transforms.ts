// eslint-disable-file @typescript-eslint/ban-ts-comment
import { IntOrString } from '@ui/gen/k8s.io/apimachinery/pkg/util/intstr/generated_pb';
import {
  AnalysisRunSpec,
  AnalysisRunStatus,
  Argument,
  CloudWatchMetric,
  DatadogMetric,
  Measurement,
  MetricProvider,
  MetricResult
} from '@ui/gen/rollouts/api/v1alpha1/generated_pb';

import {
  AnalysisStatus,
  FunctionalStatus,
  MeasurementSetInfo,
  MeasurementValueInfo,
  TransformedMeasurement,
  TransformedMetric,
  TransformedMetricSpec,
  TransformedMetricStatus,
  TransformedValueObject
} from './types';

export const isFiniteNumber = (value?: IntOrString | number | string) => {
  if (!value) {
    return false;
  }
  if (Number.isFinite((value as IntOrString).intVal)) {
    return Number.isFinite((value as IntOrString).intVal);
  }
  return Number.isFinite(Number(value.toString()));
};

export const getFiniteNumber = (
  value?: IntOrString | number | string,
  fallback?: number
): number => {
  if (!value || !isFiniteNumber(value)) {
    return fallback ? fallback : 0;
  }
  const intVal = 'intVal' in (value as IntOrString) ? (value as IntOrString).intVal : value;
  return intVal as number;
};

export const roundNumber = (value: number): number => Math.round(value * 100) / 100;

// Overall Analysis Utils

/**
 *
 * @param metricResults array of metric results
 * @returns timestamp in ms or null
 */
export const analysisEndTime = (metricResults: MetricResult[]): number => {
  if (metricResults.length === 0) {
    return 0;
  }

  const measurementEndTimes: number[] = [];
  metricResults.forEach((metricResult) => {
    (metricResult.measurements ?? []).forEach((measurement) => {
      const seconds = Number(measurement.finishedAt?.seconds);
      if (seconds > 0) {
        measurementEndTimes.push(seconds);
      }
    });
  });

  return Math.max(...measurementEndTimes);
};

// Arg Utils

/**
 *
 * @param args arguments name/value pairs associated with the analysis run
 * @param argName name of arg for which to find the value
 * @returns
 * value associated with the arg
 * or null if args is empty
 * or null if argName is not present in args
 * or null if arg value is undefined or null
 */
export const argValue = (args: Argument[], argName: string): string | null =>
  args.find((arg) => arg.name === argName)?.value ?? null;

// Metric Utils

/**
 *
 * @param providerInfo metric provider object
 * @returns first key in the provider object
 */
export const metricProvider = (providerInfo: MetricProvider): string =>
  Object.keys(providerInfo)?.[0] ?? 'unsupported provider';

const PROVIDER_CONDITION_SUPPORT: {
  [key: string]: (resultAccessor: string) => {
    isFormatSupported: boolean;
    conditionKey: string | null;
  };
} = {
  prometheus: (resultAccessor: string) => ({
    isFormatSupported: resultAccessor === 'result[0]',
    conditionKey: '0'
  }),
  datadog: (resultAccessor: string) => ({
    isFormatSupported: ['result', 'default(result, 0)'].includes(resultAccessor),
    conditionKey: resultAccessor.includes('0') ? '0' : null
  }),
  wavefront: (resultAccessor: string) => ({
    isFormatSupported: resultAccessor === 'result',
    conditionKey: null
  }),
  newRelic: (resultAccessor: string) => ({
    isFormatSupported: resultAccessor.startsWith('result.'),
    conditionKey: resultAccessor.substring(7)
  }),
  cloudWatch: () => ({
    isFormatSupported: false,
    conditionKey: null
  }),
  graphite: (resultAccessor: string) => ({
    isFormatSupported: resultAccessor === 'result[0]',
    conditionKey: '0'
  }),
  influxdb: (resultAccessor: string) => ({
    isFormatSupported: resultAccessor === 'result[0]',
    conditionKey: '0'
  }),
  skywalking: () => ({
    isFormatSupported: false,
    conditionKey: null
  })
};

/**
 *
 * @param condition failure_condition or success_condition with the format
 * [result accessor] [operator] {{ args.[argname] }}
 * or [result accessor] [operator] [value]
 * @param args arguments name/value pairs associated with the analysis run
 * @returns
 * label - a friendly fail/success condition label and
 * thresholds - threshold values that can be converted into numbers
 * conditionKeys - string keys for the values being compared in the condition
 */
export const conditionDetails = (
  condition?: string,
  args: Argument[] = [],
  provider?: MetricProvider
): {
  label: string | null;
  thresholds: number[];
  conditionKeys: string[];
} => {
  if (
    condition === undefined ||
    condition === '' ||
    provider === undefined ||
    metricProvider(provider) === 'unsupported provider'
  ) {
    return {
      label: null,
      thresholds: [],
      conditionKeys: []
    };
  }

  const interpolatedCondition = interpolateQuery(condition, args);
  const subconditions = (interpolatedCondition || '').split(/ && | \|\| /);

  const providerType = metricProvider(provider);
  const thresholds: number[] = [];
  const conditionKeys: string[] = [];

  // for each subcondition, if it deemed to be a supported subcondition, add keys and numeric thresholds
  subconditions.forEach((subcondition) => {
    const subconditionParts = subcondition.split(' ');
    if (subconditionParts.length === 3) {
      const providerInfo = PROVIDER_CONDITION_SUPPORT[providerType]?.(subconditionParts[0].trim());
      const isFormatSupported = providerInfo?.isFormatSupported ?? false;
      const conditionKey = providerInfo?.conditionKey ?? null;

      const isUnderOverThreshold =
        subconditionParts[1].includes('<') || subconditionParts[1].includes('>');
      const isChartableThreshold = isFiniteNumber(parseFloat(subconditionParts[2]));

      if (isFormatSupported && isUnderOverThreshold && isChartableThreshold) {
        if (conditionKey !== null) {
          conditionKeys.push(conditionKey);
        }
        thresholds.push(Number(subconditionParts[2]));
      }
    }
  });

  return {
    label: interpolatedCondition || null,
    thresholds,
    conditionKeys: [...new Set(conditionKeys)]
  };
};

/**
 *
 * @param thresholds threshold values
 * @returns number formatted to two decimal points
 */
export const formatThresholdsForChart = (thresholds: number[]): number[] =>
  thresholds.map((t) => roundNumber(t));

/**
 *
 * @param valueMax max value for a measurement
 * @param failThresholds fail thresholds for the metric
 * @param successThresholds success thresholds for the metric
 * @returns 120% of the max content value which could either be a data point or one of the thresholds
 * or 1 if the max value is less than 1 and there are no thresholds
 */
export const chartMax = (
  valueMax: number,
  failThresholds: number[] | null,
  successThresholds: number[] | null
) => {
  if (valueMax < 1 && failThresholds === null && successThresholds === null) {
    return 1;
  }
  const failThresholdMax =
    failThresholds !== null && failThresholds.length > 0
      ? Math.max(...failThresholds)
      : Number.NEGATIVE_INFINITY;
  const successThresholdMax =
    successThresholds !== null && successThresholds.length > 0
      ? Math.max(...successThresholds)
      : Number.NEGATIVE_INFINITY;
  return roundNumber(Math.max(valueMax, failThresholdMax, successThresholdMax) * 1.2);
};

/**
 *
 * @param phase analysis phase
 * @returns analysis phase adjusted to render the UI status with a more accurate functional status
 */
export const getAdjustedMetricPhase = (phase?: AnalysisStatus): AnalysisStatus =>
  phase === AnalysisStatus.Error ? AnalysisStatus.Failed : (phase ?? AnalysisStatus.Unknown);

/**
 *
 * @param specAndStatus analysis spec and status information
 * @returns analysis metrics with additional information to render to the UI
 */
export const transformMetrics = (
  spec?: AnalysisRunSpec,
  status?: AnalysisRunStatus
): { [key: string]: TransformedMetric } => {
  if (!spec || !status) {
    return {};
  }

  const transformedMetrics: { [key: string]: TransformedMetric } = {};
  status.metricResults?.forEach((metricResults, idx) => {
    const metricName = metricResults?.name ?? `Unknown metric ${idx}`;
    const metricSpec = spec?.metrics?.find((m) => m.name === metricName);

    if (metricSpec !== undefined) {
      // spec values
      const failConditionInfo = conditionDetails(
        metricSpec.failureCondition,
        spec.args,
        metricSpec.provider
      );
      const failThresholds =
        failConditionInfo.thresholds.length > 0
          ? formatThresholdsForChart(failConditionInfo.thresholds)
          : null;
      const successConditionInfo = conditionDetails(
        metricSpec.successCondition,
        spec.args,
        metricSpec.provider
      );
      const successThresholds =
        successConditionInfo.thresholds.length > 0
          ? formatThresholdsForChart(successConditionInfo.thresholds)
          : null;

      // value keys are needed for measurement values formatted as {key1: value1, key2: value2}
      const conditionKeys = [
        ...new Set([...failConditionInfo.conditionKeys, ...successConditionInfo.conditionKeys])
      ];

      // results values
      const transformedMeasurementInfo = transformMeasurements(
        conditionKeys,
        metricResults?.measurements
      );
      const { measurements, chartable, min, max } = transformedMeasurementInfo;

      const metricStatus = (metricResults?.phase ?? AnalysisStatus.Unknown) as AnalysisStatus;
      const measurementFailures = metricResults?.failed ?? 0;
      const measurementErrors = metricResults?.error ?? 0;
      const measurementInconclusives = metricResults?.inconclusive ?? 0;

      transformedMetrics[metricName] = {
        name: metricName,
        spec: {
          ...metricSpec,
          queries: metricQueries(metricSpec.provider, spec.args),
          failConditionLabel: failConditionInfo.label,
          failThresholds,
          successConditionLabel: successConditionInfo.label,
          successThresholds,
          conditionKeys
        } as TransformedMetricSpec,
        status: {
          ...metricResults,
          adjustedPhase: getAdjustedMetricPhase(metricStatus),
          statusLabel: metricStatusLabel(
            metricStatus,
            measurementFailures,
            measurementErrors,
            measurementInconclusives
          ),
          substatus: metricSubstatus(
            metricStatus,
            measurementFailures,
            measurementErrors,
            measurementInconclusives
          ),
          transformedMeasurements: measurements,
          chartable,
          chartMin: min,
          chartMax: chartMax(max || 0, failThresholds, successThresholds)
        } as TransformedMetricStatus
      };
    }
  });

  return transformedMetrics;
};

export const analysisSubstatus = (status?: AnalysisRunStatus) => {
  return status
    ? metricSubstatus(
        status.phase as AnalysisStatus,
        status.runSummary?.failed ?? 0,
        status.runSummary?.error ?? 0,
        status.runSummary?.inconclusive ?? 0
      )
    : undefined;
};

/**
 *
 * @param status analysis metric status
 * @param failures number of measurement failures
 * @param errors number of measurement errors
 * @param inconclusives number of inconclusive measurements
 * @returns ui state substatus to indicate that there were errors/failures/
 * inconclusives
 */
export const metricSubstatus = (
  status: AnalysisStatus,
  failures: number,
  errors: number,
  inconclusives: number
): FunctionalStatus.ERROR | FunctionalStatus.WARNING | undefined => {
  switch (status) {
    case AnalysisStatus.Pending:
    case AnalysisStatus.Failed:
    case AnalysisStatus.Inconclusive:
    case AnalysisStatus.Error:
      return undefined;
    case AnalysisStatus.Running:
    case AnalysisStatus.Successful:
      if (failures > 0) {
        return FunctionalStatus.ERROR;
      }
      if (errors > 0 || inconclusives > 0) {
        return FunctionalStatus.WARNING;
      }
      return undefined;
    default:
      return undefined;
  }
};

export const analysisStatusLabel = (status?: AnalysisRunStatus) => {
  return metricStatusLabel(
    (status?.phase ?? AnalysisStatus.Unknown) as AnalysisStatus,
    status?.runSummary?.failed ?? 0,
    status?.runSummary?.error ?? 0,
    status?.runSummary?.inconclusive ?? 0
  );
};

/**
 *
 * @param status analysis metric status
 * @param failures number of measurement failures
 * @param errors number of measurement errors
 * @param inconclusives number of inconclusive measurements
 * @returns descriptive label to include more details beyond the overall
 * analysis status
 */
export const metricStatusLabel = (
  status: AnalysisStatus,
  failures: number,
  errors: number,
  inconclusives: number
) => {
  let extraDetails = '';
  const hasFailures = failures > 0;
  const hasErrors = errors > 0;
  const hasInconclusives = inconclusives > 0;
  switch (status) {
    case AnalysisStatus.Unknown:
      return 'Analysis status unknown';
    case AnalysisStatus.Pending:
      return 'Analysis pending';
    case AnalysisStatus.Running:
      return 'Analysis in progress';
    case AnalysisStatus.Failed:
      return `Analysis failed`;
    case AnalysisStatus.Inconclusive:
      return `Analysis inconclusive`;
    case AnalysisStatus.Error:
      return 'Analysis errored';
    case AnalysisStatus.Successful:
      if (hasFailures && !hasErrors && !hasInconclusives) {
        extraDetails = 'with measurement failures';
      } else if (!hasFailures && hasErrors && !hasInconclusives) {
        extraDetails = 'with measurement errors';
      } else if (!hasFailures && !hasErrors && hasInconclusives) {
        extraDetails = 'with inconclusive measurements';
      } else if (hasFailures || hasErrors || hasInconclusives) {
        extraDetails = 'with multiple issues';
      }
      return `Analysis passed ${extraDetails}`.trim();
    default:
      return '';
  }
};

/**
 *
 * @param query query for an analysis run metric
 * @param args arguments name/value pairs associated with the analysis run
 * @returns the query with all {{ args.[argName] }} replaced with
 * the value of the arg
 */
export const interpolateQuery = (query?: string, args?: Argument[]) => {
  if (!query) {
    return;
  }
  if (!args || args.length === 0) {
    return query;
  }

  const regex = /\{{.*?\}}/g;
  return query.replace(regex, (match) => {
    const argPieces = match.replace(/[{ }]/g, '').split('.');
    const replacementValue = argValue(args, argPieces?.[1] ?? '');
    return replacementValue ?? match;
  });
};

const interpolatedQueryArray = (query?: string, args?: Argument[]): string[] => {
  const interpolated = interpolateQuery(query, args);
  if (interpolated) {
    return [interpolated];
  }
  return [];
};

/**
 *
 * @param datadog datadog metric object
 * @param args arguments name/value pairs associated with the analysis run
 * @returns query formatted for display
 */
export const printableDatadogQuery = (datadog?: DatadogMetric, args?: Argument[]): string[] => {
  if (datadog && (datadog?.apiVersion ?? '').toLowerCase() === 'v1' && 'query' in datadog) {
    return interpolatedQueryArray(datadog.query, args);
  }
  if (datadog && (datadog?.apiVersion ?? '').toLowerCase() === 'v2') {
    if ('query' in datadog) {
      return 'formula' in datadog
        ? [`query: ${interpolateQuery(datadog.query, args)}, formula: ${datadog.formula}`]
        : interpolatedQueryArray(datadog.query, args);
    }
    if ('queries' in datadog) {
      const interpolatedQueries: { [key: string]: string } = {};
      Object.keys(datadog.queries).forEach((queryKey) => {
        const interpolated = interpolateQuery(datadog.queries[queryKey], args);
        if (interpolated) {
          interpolatedQueries[queryKey] = interpolated;
        }
      });
      return 'formula' in datadog
        ? [`queries: ${JSON.stringify(interpolatedQueries)}, formula: ${datadog.formula}`]
        : Object.values(datadog.queries).reduce((result, query) => {
            const interpolated = interpolateQuery(query, args);
            if (interpolated) {
              result.push(interpolated);
            }
            return result;
          }, [] as string[]);
    }
  }
  return [];
};

/**
 *
 * @param cloudWatch cloudwatch metric object
 * @returns query formatted for display or undefined
 */
export const printableCloudWatchQuery = (cloudWatch?: CloudWatchMetric): string[] | undefined => {
  return Array.isArray(cloudWatch?.metricDataQueries)
    ? cloudWatch.metricDataQueries.map((query) => JSON.stringify(query))
    : undefined;
};

/**
 *
 * @param provider metric provider object
 * @param args arguments name/value pairs associated with the analysis run
 * @returns query formatted for display or undefined
 */
export const metricQueries = (
  provider?: MetricProvider,
  args: Argument[] = []
): string[] | undefined => {
  if (provider === undefined || provider === null) {
    return undefined;
  }
  const providerType = metricProvider(provider);
  switch (providerType) {
    case 'prometheus':
      return interpolatedQueryArray(provider?.prometheus?.query, args);
    case 'datadog':
      return printableDatadogQuery(provider.datadog, args);
    case 'wavefront':
      return interpolatedQueryArray(provider?.wavefront?.query, args);
    case 'newRelic':
      return interpolatedQueryArray(provider?.newRelic?.query, args);
    case 'cloudWatch':
      return printableCloudWatchQuery(provider?.cloudWatch);
    case 'graphite':
      return interpolatedQueryArray(provider?.graphite?.query, args);
    case 'influxdb':
      return interpolatedQueryArray(provider?.influxdb?.query, args);
    case 'skywalking':
      return interpolatedQueryArray(provider?.skywalking?.query, args);
    // not currently supported: kayenta, web, job, plugin
    default:
      return undefined;
  }
};

// Measurement Utils

/**
 *
 * @param conditionKeys keys from success/fail conditions used in some cases to pull values from the measurement result
 * @param measurements array of metric measurements
 * @returns formatted measurement values and chart information if the metric can be charted
 */
export const transformMeasurements = (
  conditionKeys: string[],
  measurements?: Measurement[]
): MeasurementSetInfo => {
  if (measurements === undefined || measurements.length === 0) {
    return {
      chartable: false,
      min: 0,
      max: null,
      measurements: []
    };
  }

  return measurements.reduce(
    (
      acc: {
        chartable: boolean;
        min: number;
        max: number | null;
        measurements: TransformedMeasurement[];
      },
      currMeasurement: Measurement
    ) => {
      const transformedValue = transformMeasurementValue(conditionKeys, currMeasurement.value);
      const { canChart, tableValue } = transformedValue;
      const canCompareToBoundaries =
        canChart &&
        transformedValue.chartValue !== null &&
        isFiniteNumber(transformedValue.chartValue as number);

      return {
        chartable: acc.chartable && canChart,
        min: canCompareToBoundaries
          ? Math.min(Number(transformedValue.chartValue), acc.min)
          : acc.min,
        max: canCompareToBoundaries
          ? Math.max(Number(transformedValue.chartValue), acc.max ?? 0)
          : acc.max,
        measurements: [
          ...acc.measurements,
          {
            ...currMeasurement,
            chartValue: transformedValue.chartValue,
            tableValue
          }
        ]
      } as {
        chartable: boolean;
        min: number;
        max: number | null;
        measurements: TransformedMeasurement[];
      };
    },
    { chartable: true, min: 0, max: null, measurements: [] as TransformedMeasurement[] }
  );
};

/**
 *
 * @param value value to check for chartability
 * @returns whether the data point can be added to a line chart (number or null)
 */
export const isChartable = (value: number): boolean => isFiniteNumber(value) || value === null;

type FormattedMeasurementValue = number | string | null;

/**
 *
 * @param value value to display
 * @returns value formatted for display purposes
 */
export const formattedValue = (value: number): FormattedMeasurementValue => {
  const isNum = isFiniteNumber(value);
  return isNum ? roundNumber(Number(value)) : (value?.toString() ?? null);
};

/**
 *
 * @param value measurement value number (examples: 4 or 4.05)
 * @returns information about displaying the measurement value
 */
const formatNumberMeasurement = (value: number): MeasurementValueInfo => {
  const displayValue = formattedValue(value);
  return {
    canChart: true,
    chartValue: displayValue,
    tableValue: displayValue
  };
};

/**
 *
 * @param value measurement value array (examples: [4] or [null] or ['anything else'])
 * @param accessor key by which to access measurement value
 * @returns information about displaying the measurement value
 */
export const formatSingleItemArrayMeasurement = (
  value: FormattedMeasurementValue[],
  accessor: number
): MeasurementValueInfo => {
  if (isFiniteNumber(accessor)) {
    const measurementValue = value?.[accessor] as number;
    // if it's a number or null, chart it
    if (isFiniteNumber(measurementValue as number) || measurementValue === null) {
      const displayValue = formattedValue(measurementValue);
      return {
        canChart: isChartable(measurementValue),
        chartValue: { [accessor]: displayValue },
        tableValue: { [accessor]: displayValue }
      };
    }
    // if it exists, but it's not a good format, just put it in a table
    return {
      canChart: false,
      tableValue: { [accessor]: measurementValue.toString() }
    };
  }
  return {
    canChart: false,
    tableValue: value.toString()
  };
};

/**
 *
 * @param value measurement value array (examples: [4,6,3,5] or [4,6,null,5] or [4,6,'a string',5])
 * @returns information about displaying the measurement value (charts a chartable first value, shows stringified value in table))
 */
export const formatMultiItemArrayMeasurement = (
  value: FormattedMeasurementValue[]
): MeasurementValueInfo => {
  if (value.length === 0) {
    return {
      canChart: false,
      tableValue: ''
    };
  }

  const firstMeasurementValue = value[0] as number;
  const canChartFirstValue = isChartable(firstMeasurementValue);
  return {
    canChart: canChartFirstValue,
    ...(canChartFirstValue && { chartValue: formattedValue(firstMeasurementValue) }),
    tableValue: value.map((v) => String(v)).toString()
  };
};

/**
 *
 * @param value measurement value object (example: { key1: 5, key2: 154, key3: 'abc' }
 * @param accessors keys by which to access measurement values
 * @returns information about displaying the measurement value (returns TransformedObjectValue))
 */
export const formatKeyValueMeasurement = (
  value: { [key: string]: FormattedMeasurementValue },
  accessors: string[]
): MeasurementValueInfo => {
  const transformedValue: TransformedValueObject = {};
  let canChart = true;
  accessors.forEach((accessor) => {
    if (accessor in value) {
      const measurementValue = value[accessor] as number;
      const displayValue = formattedValue(measurementValue);
      canChart = canChart && isChartable(measurementValue);
      transformedValue[accessor] = displayValue;
    } else {
      transformedValue[accessor] = null;
    }
  });
  return {
    canChart:
      canChart &&
      !Object.values(transformedValue).every((v: FormattedMeasurementValue) => v === null),
    chartValue: transformedValue,
    tableValue: transformedValue
  };
};

/**
 *
 * @param conditionKeys keys from success/fail conditions used in some cases to pull values from the measurement result
 * @param value measurement value returned by provider
 * @returns chart and table data along with a flag indicating whether the measurement value can be charted
 */
const transformMeasurementValue = (
  conditionKeys: string[],
  value?: string
): MeasurementValueInfo => {
  if (value === undefined || value === '') {
    return {
      canChart: true,
      chartValue: null,
      tableValue: null
    };
  }

  const parsedValue = JSON.parse(value);

  // single number measurement value
  if (isFiniteNumber(parsedValue)) {
    return formatNumberMeasurement(parsedValue);
  }

  // single item array measurement value
  if (Array.isArray(parsedValue) && parsedValue.length > 0 && conditionKeys.length === 1) {
    const accessor = parseInt(conditionKeys[0]);
    return formatSingleItemArrayMeasurement(parsedValue, accessor);
  }

  // multi-item array measurement value
  if (Array.isArray(parsedValue) && parsedValue.length > 0) {
    return formatMultiItemArrayMeasurement(parsedValue);
  }

  // key / value pairs measurement value
  if (typeof parsedValue === 'object' && !Array.isArray(parsedValue) && conditionKeys.length > 0) {
    return formatKeyValueMeasurement(parsedValue, conditionKeys);
  }

  // unsupported formats are stringified and put into table
  return {
    canChart: false,
    tableValue: parsedValue.toString()
  };
};
